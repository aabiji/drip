package p2p

import (
	"log"
	"os"
	"path"
	"sync"
	"syscall"

	"github.com/edsrzf/mmap-go"
)

type TransferChunk struct {
	TransferId string  `json:"transferId"`
	Recipient  string  `json:"recipient"`
	Data       []uint8 `json:"data"`
	Offset     int64   `json:"offset"`
}

type Transfer struct {
	SessionId  string `json:"sessionId"`
	TransferId string `json:"transferId"`
	Recipient  string `json:"recipient"`
	FileSize   int64  `json:"size"`

	file   mmap.MMap
	closed bool
	done   bool
}

type SessionInfo struct {
	SessionId  string     `json:"sessionId"`
	Recipients []string   `json:"recipients"`
	Transfers  []Transfer `json:"transfers"`
	Sender     string     `json:"sender,omitempty"`
}

type SessionCancel struct {
	SessionId  string   `json:"sessionId"`
	Recipients []string `json:"recipients"`
}

type SessionResponse struct {
	SessionId string `json:"sessionId"`
	Accepted  bool   `json:"accepted"`
}

func (t *Transfer) Close() {
	if !t.closed {
		if err := t.file.Unmap(); err != nil {
			panic(err)
		}
		t.closed = true
	}
}

type Downloader struct {
	transfers map[string]*Transfer
	mutex     sync.Mutex // guards transfers
	sessions  map[string]SessionInfo

	downloadFolder    *string
	authorizeCallback func(string) bool
	notifyCallback    func(string, int)
}

func NewDownloader(
	downloadFolder *string,
	authorizeCallback func(string) bool,
	notifyCallback func(string, int),
) *Downloader {
	return &Downloader{
		transfers: make(map[string]*Transfer),
		sessions:  make(map[string]SessionInfo),

		downloadFolder:    downloadFolder,
		authorizeCallback: authorizeCallback,
		notifyCallback:    notifyCallback,
	}
}

func (d *Downloader) Close() {
	for _, transfer := range d.transfers {
		transfer.file.Flush()
		transfer.Close()
	}
}

func (d *Downloader) CancelSessions(disconnectedPeer string) {
	for _, session := range d.sessions {
		if session.Sender == disconnectedPeer {
			d.receiveCancel(session.SessionId)
		}
	}
}

func (d *Downloader) ReceiveMessage(msg Message) *Message {
	switch msg.MessageType {
	case SESSION_INFO:
		info, err := DeserializeInto[SessionInfo](msg)
		if err != nil {
			panic(err)
		}
		info.Sender = msg.SenderId
		return d.receiveInfo(info)
	case TRANSFER_CHUNK:
		chunk, err := DeserializeInto[TransferChunk](msg)
		if err != nil {
			panic(err)
		}
		return d.receiveChunk(chunk)
	case SESSION_CANCEL:
		info, err := DeserializeInto[SessionCancel](msg)
		if err != nil {
			panic(err)
		}
		return d.receiveCancel(info.SessionId)
	}
	log.Panicf("Received unkonwn message: %v\n", msg)
	return nil
}

// linux specific syscall to allocate the size of a file
// TODO: implement version for other operating systems
func fallocate(file *os.File, offset int64, length int64) error {
	if length == 0 {
		return nil
	}
	return syscall.Fallocate(int(file.Fd()), 0, offset, length)
}

func OpenFile(path string, size int64) (mmap.MMap, error) {
	// TODO: should be able to create the file with
	// the same file permissions as the sender -- how to do in os-agnostic way?
	// the permission should be applied after all the file contents have been recived
	exists := true
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if os.IsNotExist(err) {
		exists = false
	} else if err != nil {
		return nil, err
	}
	defer file.Close()

	if !exists {
		err = fallocate(file, 0, size)
		if err != nil {
			return nil, err
		}
	}

	fileData, err := mmap.Map(file, mmap.RDWR, 0)
	return fileData, err
}

func (d *Downloader) receiveInfo(info SessionInfo) *Message {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	response := SessionResponse{SessionId: info.SessionId, Accepted: false}
	authorized := d.authorizeCallback(info.Sender)
	if !authorized {
		msg := NewMessage(string(SESSSION_RESPONSE), response)
		return &msg
	}

	d.sessions[info.SessionId] = info
	for _, t := range info.Transfers {
		if t.Recipient != getDeviceName() {
			continue // not for us
		}

		fullpath := path.Join(*d.downloadFolder, t.TransferId)
		contents, err := OpenFile(fullpath, t.FileSize)
		if err != nil {
			panic(err)
		}

		t.file = contents
		d.transfers[t.TransferId] = &t
	}

	response.Accepted = true
	msg := NewMessage(string(SESSSION_RESPONSE), response)
	return &msg
}

func (d *Downloader) handleTransferCompletion(transfer *Transfer) {
	transfer.Close()
	transfer.done = true

	// check if all the tranfers associated to a session are complete
	allDone := true
	session := d.sessions[transfer.SessionId]
	for _, transferCopy := range session.Transfers {
		if transferCopy.Recipient != getDeviceName() {
			continue // not for us
		}

		t := d.transfers[transferCopy.TransferId]
		if !t.done {
			allDone = false
			break
		}
	}

	if allDone {
		// remove the session data from cache
		for id := range d.transfers {
			delete(d.transfers, id)
		}
		delete(d.sessions, session.SessionId)

		// Notify the user
		d.notifyCallback(session.Sender, len(session.Transfers))
	}
}

func (d *Downloader) receiveChunk(chunk TransferChunk) *Message {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	transfer, exists := d.transfers[chunk.TransferId]
	if !exists {
		return nil // we must have deleted the transfer beforehand
	}
	chunkSize := int64(len(chunk.Data))

	if err := transfer.file.Lock(); err != nil {
		panic(err)
	}
	copy(transfer.file[chunk.Offset:], chunk.Data)
	if err := transfer.file.Unlock(); err != nil {
		panic(err)
	}
	if err := transfer.file.Flush(); err != nil { // TODO: faster way?
		panic(err)
	}

	// last chunk, done writing
	if chunk.Offset+chunkSize >= transfer.FileSize {
		d.handleTransferCompletion(transfer)
	}
	return nil
}

func (d *Downloader) receiveCancel(sessionId string) *Message {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for _, transferCopy := range d.sessions[sessionId].Transfers {
		if transferCopy.Recipient != getDeviceName() {
			continue // not for us
		}

		transfer := d.transfers[transferCopy.TransferId]
		transfer.Close()

		fullpath := path.Join(*d.downloadFolder, transfer.TransferId)
		if err := os.Remove(fullpath); err != nil {
			panic(err)
		}

		delete(d.transfers, transfer.TransferId)
	}
	delete(d.sessions, sessionId)
	return nil
}
