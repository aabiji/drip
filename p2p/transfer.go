package p2p

import "encoding/json"

// file transfer message types
const (
	TRANSFER_START = 0
	TRANSFER_CHUNK = 1
	TRANSFER_ACK   = 2 // acknowledgement
)

type TransferInfo struct {
	Recipients []string `json:"recipients"`
	FileName   string   `json:"name"`
	FileSize   int64    `json:"size"`
	NumChunks  int      `json:"numChunks"`
}

type FileChunk struct {
	Data       []uint8  `json:"data"`
	Index      int      `json:"chunkIndex"`
	Recipients []string `json:"recipients"`
}

type Acknowledgement struct { // TODO; shorter name for this
	chunksReceived []bool `json:"received"`
}

type TransferService struct { // TODO: better variable name for this
	// pending messages to be sent through the peer's data channel
	Pending map[string]chan Message
}

func NewTransferService() TransferService {
	return TransferService{}
}

func (t *TransferService) AddPeer(id string) {
	t.Pending[id] = make(chan Message, 100)
}

func (t *TransferService) HandleMessage(msg Message) Message {
	switch msg.MessageType {
	case TRANSFER_START:
		var info TransferInfo
		err := json.Unmarshal(msg.Data, &info)
		if err != nil {
			panic(err)
		}
		return t.handleInfo(info)
	case TRANSFER_CHUNK:
		var chunk FileChunk
		err := json.Unmarshal(msg.Data, &chunk)
		if err != nil {
			panic(err)
		}
		return t.handleChunk(chunk)
	case TRANSFER_ACK:
		var ack Acknowledgement
		err := json.Unmarshal(msg.Data, &ack)
		if err != nil {
			panic(err)
		}
		return t.handleAck(ack)
	}
}

// TODO!
func (t *TransferService) handleInfo(info TransferInfo) Message {

}

func (t *TransferService) handleChunk(chunk FileChunk) Message {

}

func (t *TransferService) handleAck(ack Acknowledgement) Message {

}
