package p2p

import (
	"encoding/json"
)

type Message struct {
	MessageType int
	Data        []byte
}

func NewMessage[T any](messageType int, value T) Message {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return Message{MessageType: messageType, Data: encoded}
}

func (m *Message) Serialize() []byte {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return bytes
}

// TODO: function to automatically deserialize based on the different types...
