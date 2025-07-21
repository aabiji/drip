package p2p

import (
	"log"
	"os"
	"path"

	"github.com/edsrzf/mmap-go"
)

type TransferChunk struct {
	TransferId string  `json:"transferId"`
	Recipient  string  `json:"recipient"`
	Data       []uint8 `json:"data"`
	Offset     int64   `json:"offset"`
}

type Transfer struct {
	TransferId string `json:"transferId"`
	Recipient  string `json:"recipient"`
	FileSize   int64  `json:"size"`

	file mmap.MMap
}

type SessionInfo struct {
	SessionId  string     `json:"sessionId"`
	Recipients []string   `json:"recipients"`
	Transfers  []Transfer `json:"transfers"`
}

type SessionCancel struct {
	SessionId  string   `json:"sessionId"`
	Recipients []string `json:"recipients"`
}

type SessionResponse struct {
	SessionId string `json:"sessionId"`
	Accepted  bool   `json:"accepted"`
}

type Downloader struct {
	downloadFolder string
	Transfers      map[string]*Transfer
	Sessions       map[string]SessionInfo
}

func NewDownloader() *Downloader {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return &Downloader{
		Transfers:      make(map[string]*Transfer),
		Sessions:       make(map[string]SessionInfo),
		downloadFolder: path.Join(home, "Downloads"),
	}
}

func (d *Downloader) Close() error {
	for _, transfer := range d.Transfers {
		transfer.file.Flush()
		transfer.file.Unmap()
	}
	return nil
}

func (d *Downloader) ReceiveMessage(msg Message) *Message {
	switch msg.MessageType {
	case SESSION_INFO:
		info, err := DeserializeInto[SessionInfo](msg)
		if err != nil {
			panic(err)
		}
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
	// TODO: ask for user confirmation

	d.Sessions[info.SessionId] = info
	for _, t := range info.Transfers {
		if t.Recipient != getDeviceName() {
			continue // not for us
		}

		fullpath := path.Join(d.downloadFolder, t.TransferId)
		contents, err := OpenFile(fullpath, t.FileSize)
		if err != nil {
			panic(err)
		}

		transfer := &Transfer{
			TransferId: t.TransferId,
			Recipient:  t.Recipient,
			FileSize:   t.FileSize,
			file:       contents,
		}
		log.Println("Set ", t.TransferId)
		d.Transfers[t.TransferId] = transfer
	}

	msg := NewMessage(SESSSION_RESPONSE, SessionResponse{
		SessionId: info.SessionId, Accepted: true,
	})
	return &msg
}

func (d *Downloader) receiveChunk(chunk TransferChunk) *Message {
	transfer := d.Transfers[chunk.TransferId]
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
		transfer.file.Unmap()
		log.Printf("Downloaded %s\n", chunk.TransferId)
	}
	return nil
}

func (d *Downloader) receiveCancel(sessionId string) *Message {
	for _, transferCopy := range d.Sessions[sessionId].Transfers {
		if transferCopy.Recipient != getDeviceName() {
			continue // not for us
		}

		transfer := d.Transfers[transferCopy.TransferId]
		if err := transfer.file.Unmap(); err != nil {
			panic(err)
		}

		fullpath := path.Join(d.downloadFolder, transfer.TransferId)
		if err := os.Remove(fullpath); err != nil {
			panic(err)
		}

		delete(d.Transfers, transfer.TransferId)
	}
	delete(d.Sessions, sessionId)
	return nil
}
