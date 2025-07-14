package p2p

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"time"

	"github.com/pion/webrtc/v4"
)

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

type TCPMedium struct {
	packets  chan Message
	peerAddr string
	ourAddr  string
}

func (t *TCPMedium) QueueMessage(msg Message) { t.packets <- msg }

func (t *TCPMedium) Close() {
	// TODO!
}

func sendTCPMessage(conn net.Conn, msg Message) error {
	data := msg.Serialize()
	length := uint32(len(data))
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, length)

	_, err := conn.Write(append(header, data...))
	return err
}

func readFramedMessage(conn net.Conn) ([]byte, error) {
	header := make([]byte, 4)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(header)
	body := make([]byte, length)
	_, err = io.ReadFull(conn, body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (t *TCPMedium) ForwardMessages() {
	var conn net.Conn
	var err error

	// keep trying to connect to peer until succesful
	for {
		conn, err = net.Dial("tcp", t.peerAddr)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
		log.Println("Retrying peer TCP connection")
	}
	defer conn.Close()

	for pkt := range t.packets {
		err := sendTCPMessage(conn, pkt)
		if err != nil {
			panic(err)
		}
	}
}

func (t *TCPMedium) ReceiveMessages(handler func(msg Message)) {
	listener, err := net.Listen("tcp", t.ourAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go func(c net.Conn) {
			defer c.Close()
			for {
				data, err := readFramedMessage(conn)
				if err != nil {
					panic(err)
				}
				msg := Deserialize(data)
				handler(msg)
			}
		}(conn)
	}
}

type WebRTCMedium struct {
	pending     chan Message
	dataChannel *webrtc.DataChannel
	connected   bool
}

func NewWebRTCMedium(channel *webrtc.DataChannel) WebRTCMedium {
	m := WebRTCMedium{
		pending:     make(chan Message, 100),
		dataChannel: channel,
		connected:   false,
	}
	m.dataChannel.OnClose(m.Close) // TODO: allow the user to pass in a close handler
	return m
}

func (w *WebRTCMedium) QueueMessage(msg Message) { w.pending <- msg }

func (w *WebRTCMedium) ForwardMessages() {
	w.dataChannel.OnOpen(func() {
		w.connected = true
		for msg := range w.pending {
			log.Printf("Sending: %v\n", msg)
			w.dataChannel.Send(msg.Serialize())
		}
	})
}

func (w *WebRTCMedium) ReceiveMessages(handler func(msg Message)) {
	w.dataChannel.OnMessage(func(channelMsg webrtc.DataChannelMessage) {
		msg := Deserialize(channelMsg.Data)
		if msg.SenderId == getDeviceName() {
			return // ignore our own messages
		}
		log.Printf("Receiving: %v\n", msg)
	})
}

func (w *WebRTCMedium) Close() {
	w.connected = false
	w.dataChannel.GracefulClose()
}
