package p2p

import (
	"log"
	"os"
	"path"
	"sync"

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
	downloadFolder string
	transfers      map[string]*Transfer
	mutex          sync.Mutex // guards transfers
	sessions       map[string]SessionInfo

	authorizeCallback func(string) bool
	notifyCallback    func(string, int)
}

func NewDownloader(
	authorizeCallback func(string) bool,
	notifyCallback func(string, int),
) *Downloader {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return &Downloader{
		transfers:      make(map[string]*Transfer),
		sessions:       make(map[string]SessionInfo),
		downloadFolder: path.Join(home, "Downloads"),

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

		fullpath := path.Join(d.downloadFolder, t.TransferId)
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

		fullpath := path.Join(d.downloadFolder, transfer.TransferId)
		if err := os.Remove(fullpath); err != nil {
			panic(err)
		}

		delete(d.transfers, transfer.TransferId)
	}
	delete(d.sessions, sessionId)
	return nil
}
