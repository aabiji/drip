package p2p

import "encoding/json"

const (
	// tcp webrtc message types
	OFFER_PACKET = iota
	ANSWER_PACKET
	ICE_PACKET

	// file transfer message types
	SESSION_INFO
	SESSSION_RESPONSE
	SESSION_CANCEL
	TRANSFER_CHUNK

	// event types
	PEERS_UPDATED
)

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

func GetMessage(bytes []byte) Message {
	m := Message{}
	err := json.Unmarshal(bytes, &m)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *Message) Serialize() []byte {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes
}

func Deserialize[T any](msg Message) (T, error) {
	var result T
	err := json.Unmarshal(msg.Data, &result)
	return result, err
}
