package p2p

import "encoding/json"

const (
	// udp webrtc message types
	OFFER_PACKET = iota
	ANSWER_PACKET
	ICE_PACKET

	// file transfer message types
	TRANSFER_START
	TRANSFER_CHUNK
	TRANSFER_REPLY
)

// file transfer message types
type FileInfo struct {
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

type Reply struct {
	ChunksReceived []bool `json:"received"`
}

// generic message type
type Message struct {
	SenderId    string
	MessageType int
	Data        []byte
}

func NewMessage[T any](messageType int, value T) Message {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return Message{
		MessageType: messageType,
		Data:        encoded,
		SenderId:    getDeviceName(),
	}
}

func (m *Message) Serialize() []byte {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes
}

func Deserialize(bytes []byte) Message {
	m := Message{}
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		panic(err)
	}
	return m
}

func DeserializeInto[T any](msg Message) (T, error) {
	var result T
	err := json.Unmarshal(msg.Data, &result)
	return result, err
}
