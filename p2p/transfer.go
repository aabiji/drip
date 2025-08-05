package p2p

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/edsrzf/mmap-go"
	"github.com/google/uuid"
)

const ( // message types
	TRANSFER_CHUNK = iota
	TRANSFER_CANCELLED
	TRANSFER_INFO
	TRANSFER_REQUEST
	TRANSFER_RESPONSE
)

type Transfer struct {
	Sender     string
	Id         string
	Recipients []string
	Files      map[string]*File

	pending              bool
	authorizedRecipients []string
}

type TransferRequest struct {
	Sender     string
	TransferId string
	Message    string
}

type TransferResponse struct {
	TransferId string
	Authorized bool
}

type File struct {
	Name string
	Size int64

	reader     io.ReadCloser
	amountSent int64

	writer        mmap.MMap
	doneReceiving bool

	ctx    context.Context
	cancel context.CancelFunc
}

type Chunk struct {
	TransferId string
	Filename   string
	Offset     int64
	Data       []byte
}

func NewReaderFile(name string, size int64, rc io.ReadCloser) *File {
	ctx, cancel := context.WithCancel(context.Background())
	return &File{Name: name, Size: size, reader: rc, ctx: ctx, cancel: cancel}
}

func NewWriterFile(path string, size int64) *File {
	exists := true
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		exists = false
	} else if err != nil {
		panic(err)
	}
	defer file.Close()

	if !exists {
		err = fallocate(file, 0, size)
		if err != nil {
			panic(err)
		}
	}

	fileData, err := mmap.Map(file, mmap.RDWR, 0)
	if err != nil {
		panic(err)
	}
	return &File{Name: path, Size: size, writer: fileData}
}

func (f *File) SendChunks(sendMsg func(Message), t *Transfer) {
	defer f.reader.Close()

	for {
		buffer := make([]byte, 256*1024)
		n, err := f.reader.Read(buffer)

		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		chunk := Chunk{
			TransferId: t.Id,
			Filename:   f.Name,
			Offset:     f.amountSent,
			Data:       buffer}
		f.amountSent += int64(n)

		msg := NewMessage(TRANSFER_CHUNK, chunk)
		msg.Recipients = t.Recipients
		sendMsg(msg)

		select {
		case <-f.ctx.Done():
			return
		default:
		}
	}
}

func (f *File) CloseWriter() {
	f.writer.Flush()
	f.writer.Unmap()
}

func (t Transfer) Cancel(reason string) Message {
	for _, file := range t.Files {
		file.cancel()
	}
	msg := NewMessage(TRANSFER_CANCELLED, deviceName())
	msg.Recipients = t.Recipients
	return msg
}

func (t *Transfer) handleRecipientResponse(
	authorized bool, recipient string, sendMsg func(Message)) {
	if !authorized {
		reason := fmt.Sprintf("%s has refused the transfer", recipient)
		sendMsg(t.Cancel(reason))
		return
	}

	t.authorizedRecipients = append(t.authorizedRecipients, recipient)
	if len(t.authorizedRecipients) == len(t.Recipients) {
		// got authorization from all the recipients, start sending files...
		sendMsg(NewMessage(TRANSFER_INFO, *t))

		t.pending = false
		for _, file := range t.Files {
			file.SendChunks(sendMsg, t)
		}
	}
}

type Sender struct {
	transfers map[string]Transfer
}

func NewSender() Sender {
	return Sender{transfers: make(map[string]Transfer)}
}

func (s *Sender) StartTransfer(
	recipients []string, files map[string]*File, sendMsg func(Message)) {
	id := uuid.NewString()
	s.transfers[id] = Transfer{
		Sender:     deviceName(),
		Id:         id,
		Recipients: recipients,
		Files:      files,
		pending:    true,
	}
	request := TransferRequest{
		Sender:     deviceName(),
		TransferId: id,
		Message:    fmt.Sprintf("Accept files from %s", deviceName())}
	msg := NewMessage(TRANSFER_REQUEST, request)
	msg.Recipients = recipients
	sendMsg(msg)
}

func (s *Sender) CancelTransfer(id string, sendMsg func(Message)) {
	reason := fmt.Sprintf("%s cancelled the transfer", deviceName())
	sendMsg(s.transfers[id].Cancel(reason))
}

func (n *Sender) HandleTransferResponse(
	recipient string, response TransferResponse, sendMsg func(Message)) {
	t := n.transfers[response.TransferId]
	t.handleRecipientResponse(response.Authorized, recipient, sendMsg)
}

type Receiver struct {
	transfers      map[string]Transfer
	mutex          sync.Mutex
	downloadFolder *string
	appEvents      chan Message
}

func NewReceiver(downloadFolder *string, appEvents chan Message) Receiver {
	return Receiver{
		transfers:      make(map[string]Transfer),
		downloadFolder: downloadFolder,
		appEvents:      appEvents,
	}
}

func (r *Receiver) Close() {
	for _, transfer := range r.transfers {
		for _, file := range transfer.Files {
			file.CloseWriter()
		}
	}
}

func (r *Receiver) Cancel(disconnectedPeer string) {
	for _, t := range r.transfers {
		if t.Sender == disconnectedPeer {
			r.HandleCancel(t.Id)
		}
	}
}

func (r *Receiver) HandleCancel(transferId string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, file := range r.transfers[transferId].Files {
		file.CloseWriter()
		if err := os.Remove(file.Name); err != nil {
			panic(err)
		}
	}
	delete(r.transfers, transferId)
}

func (r *Receiver) HandleInfo(transfer Transfer) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.transfers[transfer.Id] = transfer
	for _, f := range transfer.Files {
		path := path.Join(*r.downloadFolder, f.Name)
		r.transfers[transfer.Id].Files[f.Name] = NewWriterFile(path, f.Size)
	}
}

func (r *Receiver) handleTransferCompletion(id string) {
	allDone := true
	t := r.transfers[id]
	for _, file := range t.Files {
		if !file.doneReceiving {
			allDone = false
			break
		}
	}

	if allDone {
		delete(r.transfers, id)
		str := fmt.Sprintf("Received %d from %s", len(t.Files), t.Sender)
		r.appEvents <- NewMessage(NOTIFY_COMPLETION, str)
	}
}

func (r *Receiver) HandleChunk(chunk Chunk) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	file, exists := r.transfers[chunk.TransferId].Files[chunk.Filename]
	if !exists {
		return
	}
	chunkSize := int64(len(chunk.Data))

	if err := file.writer.Lock(); err != nil {
		panic(err)
	}
	copy(file.writer[chunk.Offset:], chunk.Data)
	if err := file.writer.Unlock(); err != nil {
		panic(err)
	}

	doneWriting := chunk.Offset+chunkSize >= file.Size
	if doneWriting {
		file.doneReceiving = true
		file.CloseWriter()
		r.handleTransferCompletion(chunk.TransferId)
	}
}
