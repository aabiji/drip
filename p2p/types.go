package p2p

import "github.com/edsrzf/mmap-go"

type AuthorizeCallback func(peerId string) bool
type NotifyCallback func(peerId string, numFiles int)

type ReplyHandler func(reply *Message)
type TransferMessageHandler func(msg Message, handler ReplyHandler)

type TransferChunk struct {
	TransferId string
	Recipient  string
	Data       []uint8
	Offset     int64
}

type Transfer struct {
	SessionId  string
	TransferId string
	Recipient  string
	FileSize   int64

	file   mmap.MMap
	closed bool
	done   bool
}

type SessionInfo struct {
	SessionId  string
	Recipients []string
	Transfers  []Transfer
	Sender     string
}

type SessionCancel struct {
	SessionId  string
	Recipients []string
}

type SessionResponse struct {
	SessionId string
	Accepted  bool
}

func (t *Transfer) Close() {
	if !t.closed {
		if err := t.file.Unmap(); err != nil {
			panic(err)
		}
		t.closed = true
	}
}
