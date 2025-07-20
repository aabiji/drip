package p2p

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"time"

	"github.com/pion/webrtc/v4"
)

const (
	// tcp webrtc message types
	OFFER_PACKET  = "OFFER_PACKET"
	ANSWER_PACKET = "ANSWER_PACKET"
	ICE_PACKET    = "ICE_PACKET"

	// file transfer message types
	TRANSFER_INFO     = "TRANSFER_INFO"
	TRANSFER_CHUNK    = "TRANSFER_CHUNK"
	TRANSFER_RESPONSE = "TRANSFER_RESPONSE"
	TRANSFER_CANCEL   = "TRANSFER_CANCEL"

	// general message type
	PEERS_UPDATED = "PEERS_UPDATED"
)

// generic message type
type Message struct {
	SenderId    string `json:"senderId"`
	MessageType string `json:"messageType"`
	Data        []byte `json:"data"`
}

func NewMessage[T any](messageType string, value T) Message {
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

type Medium interface {
	QueueMessage(msg Message)
	ForwardMessages(ctx context.Context)
	ReceiveMessages(ctx context.Context, handler func(msg Message))
	Connected() bool
	Close()
}

type TCPMedium struct {
	packets  chan Message
	closed   bool
	peerAddr string
	ourAddr  string
}

func NewTCPMedium(ourAddr string, peerAddr string) *TCPMedium {
	return &TCPMedium{
		packets:  make(chan Message, 25),
		closed:   false,
		peerAddr: peerAddr,
		ourAddr:  ourAddr,
	}
}

func (t *TCPMedium) QueueMessage(msg Message) { t.packets <- msg }
func (t *TCPMedium) Connected() bool          { return false } // placeholder

func (t *TCPMedium) Close() {
	if !t.closed {
		close(t.packets)
		t.closed = true
	}
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

func (t *TCPMedium) ForwardMessages(ctx context.Context) {
	var conn net.Conn
	var err error

	// keep trying to connect to peer until succesful
dialoop:
	for {
		select {
		case <-ctx.Done():
			t.Close()
			return // quit
		default:
			conn, err = net.Dial("tcp", t.peerAddr)
			if err == nil {
				break dialoop
			}
			time.Sleep(2 * time.Second)
			log.Println("Retrying peer TCP connection")
		}
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			t.Close()
			return // quit
		case pkt, ok := <-t.packets:
			if !ok {
				return
			}
			err := sendTCPMessage(conn, pkt)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (t *TCPMedium) ReceiveMessages(ctx context.Context, handler func(msg Message)) {
	listener, err := net.Listen("tcp", t.ourAddr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	for {
		// Create a channel to receive accepted connections
		connCh := make(chan net.Conn)
		go func() {
			conn, err := listener.Accept()
			if errors.Is(err, net.ErrClosed) {
				return // listneer was closed already
			} else if err != nil {
				panic(err)
			}
			connCh <- conn
		}()

		select {
		case <-ctx.Done():
			return
		case conn := <-connCh:
			go func(c net.Conn) {
				defer c.Close()
				for {
					select {
					case <-ctx.Done():
						t.Close()
						return
					default:
						data, err := readFramedMessage(c)
						if err == io.EOF {
							return
						} else if err != nil {
							panic(err)
						}
						msg := Deserialize(data)
						handler(msg)
					}
				}
			}(conn)
		}
	}
}

type WebRTCMedium struct {
	pending     chan Message
	dataChannel *webrtc.DataChannel
	connected   bool
}

func (w *WebRTCMedium) Connected() bool { return w.connected }
func (w *WebRTCMedium) Close()          { w.dataChannel.GracefulClose() }

func NewWebRTCMedium(channel *webrtc.DataChannel) *WebRTCMedium {
	m := &WebRTCMedium{
		pending:     make(chan Message, 100),
		dataChannel: channel,
		connected:   false,
	}
	m.dataChannel.OnClose(func() {
		close(m.pending)
		m.connected = false
		log.Println("Closing the data channel...")
	})
	return m
}

func (w *WebRTCMedium) QueueMessage(msg Message) { w.pending <- msg }

func (w *WebRTCMedium) ForwardMessages(ctx context.Context) {
	w.dataChannel.OnOpen(func() {
		w.connected = true
		for {
			select {
			case <-ctx.Done():
				w.dataChannel.GracefulClose()
				return
			case msg, ok := <-w.pending:
				if !ok {
					return
				}
				w.dataChannel.Send(msg.Serialize())
			}
		}
	})
}

func (w *WebRTCMedium) ReceiveMessages(ctx context.Context, handler func(msg Message)) {
	w.dataChannel.OnMessage(func(channelMsg webrtc.DataChannelMessage) {
		select {
		case <-ctx.Done():
			w.dataChannel.GracefulClose()
			return
		default:
			msg := Deserialize(channelMsg.Data)
			if msg.SenderId == getDeviceName() {
				return // ignore our own messages
			}
			handler(msg)
		}
	})
}
