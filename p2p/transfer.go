package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/edsrzf/mmap-go"
	"github.com/google/uuid"
)

const ( // message types
	TRANSFER_CHUNK = iota + 100
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

type ProgressReport struct {
	Percentages map[string]float32
	Done        bool
	Started     bool
}

func NewReaderFile(name string, size int64, rc io.ReadCloser) *File {
	ctx, cancel := context.WithCancel(context.Background())
	return &File{Name: name, Size: size, reader: rc, ctx: ctx, cancel: cancel}
}

func NewWriterFile(path string, size int64) *File {
	_, err := os.Stat(path)
	newFile := errors.Is(err, os.ErrNotExist)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	if newFile {
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
		select {
		case <-f.ctx.Done():
			return
		default:
		}

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
			Data:       buffer[:n]}
		f.amountSent += int64(n)

		msg := NewMessage(TRANSFER_CHUNK, chunk)
		msg.Recipients = t.Recipients
		sendMsg(msg)
	}
}

func (f *File) CloseWriter() {
	f.writer.Flush()
	f.writer.Unmap()
}

func (t *Transfer) Cancel() Message {
	for _, file := range t.Files {
		file.cancel()
	}
	msg := NewMessage(TRANSFER_CANCELLED, t.Id)
	msg.Recipients = t.Recipients
	return msg
}

func (t *Transfer) handleRecipientResponse(
	authorized bool, recipient string, sendMsg func(Message)) {
	if !authorized {
		sendMsg(t.Cancel())
		return
	}

	t.authorizedRecipients = append(t.authorizedRecipients, recipient)
	if len(t.authorizedRecipients) == len(t.Recipients) {
		// got authorization from all the recipients, start sending files...
		msg := NewMessage(TRANSFER_INFO, *t)
		msg.Recipients = t.Recipients
		sendMsg(msg)

		for _, file := range t.Files {
			go file.SendChunks(sendMsg, t)
		}
	}
}

type Sender struct {
	transfers map[string]*Transfer
}

func NewSender() Sender {
	return Sender{transfers: make(map[string]*Transfer)}
}

func (s *Sender) StartTransfer(
	recipients []string, files map[string]*File, sendMsg func(Message)) string {
	id := uuid.NewString()
	s.transfers[id] = &Transfer{
		Sender:     deviceName(),
		Id:         id,
		Recipients: recipients,
		Files:      files,
	}
	request := TransferRequest{
		Sender:     deviceName(),
		TransferId: id,
		Message:    fmt.Sprintf("Accept files from %s?", deviceName())}
	msg := NewMessage(TRANSFER_REQUEST, request)
	msg.Recipients = recipients
	sendMsg(msg)
	return id
}

func (s *Sender) CancelTransfer(id string, sendMsg func(Message)) {
	_, exists := s.transfers[id]
	if !exists {
		return
	}
	sendMsg(s.transfers[id].Cancel())
	delete(s.transfers, id)
}

func (s *Sender) HandleTransferResponse(
	recipient string, response TransferResponse, sendMsg func(Message)) {
	_, exists := s.transfers[response.TransferId]
	if !exists {
		return
	}

	t := s.transfers[response.TransferId]
	t.handleRecipientResponse(response.Authorized, recipient, sendMsg)
}

func (s *Sender) GetProgressReport(transferId string) ProgressReport {
	report := ProgressReport{Percentages: make(map[string]float32)}

	_, exists := s.transfers[transferId]
	if !exists {
		return report
	}

	report.Done = true
	report.Started = true
	for _, f := range s.transfers[transferId].Files {
		p := float32(float64(f.amountSent) / float64(f.Size))
		report.Percentages[f.Name] = p

		if p < 0.0001 {
			report.Started = false
		}
		if p < 1.0 {
			report.Done = false
		}
	}
	return report
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
	_, exists := r.transfers[transferId]
	if !exists {
		return
	}

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
	_, exists := r.transfers[chunk.TransferId]
	if !exists {
		return
	}

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
