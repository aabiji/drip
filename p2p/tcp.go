package p2p

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

const ( // message types
	OFFER_TCP_PACKET = iota
	ANSWER_TCP_PACKET
	ICE_TCP_PACKET
)

type TcpServer struct {
	packets  chan Message
	closed   bool
	peerAddr string
	ourAddr  string
	ctx      context.Context
}

func NewTcpServer(
	ourAddr string, peerAddr string,
	ctx context.Context,
) TcpServer {
	return TcpServer{
		packets:  make(chan Message, 25),
		closed:   false,
		peerAddr: peerAddr,
		ourAddr:  ourAddr,
		ctx:      ctx,
	}
}

func (t *TcpServer) QueueMessage(msg Message) { t.packets <- msg }

func (t *TcpServer) Close() {
	if !t.closed {
		close(t.packets)
		t.closed = true
	}
}

func sendFramedMessage(conn net.Conn, msg Message) error {
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

func (t *TcpServer) ForwardMessages() {
	var conn net.Conn
	var err error

	// keep trying to connect to peer until succesful
dialoop:
	for {
		select {
		case <-t.ctx.Done():
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
		case <-t.ctx.Done():
			t.Close()
			return // quit
		case pkt, ok := <-t.packets:
			if !ok {
				return
			}
			err := sendFramedMessage(conn, pkt)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (t *TcpServer) handleConnection(conn net.Conn, handler func(Message)) {
	defer conn.Close()
	for {
		select {
		case <-t.ctx.Done():
			t.Close()
			return
		default:
			data, err := readFramedMessage(conn)
			if err == io.EOF {
				return
			} else if err != nil {
				panic(err)
			}
			msg := GetMessage(data)
			handler(msg)
		}
	}
}

func (t *TcpServer) ReceiveMessages(handler func(Message)) {
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
		case <-t.ctx.Done():
			return
		case conn := <-connCh:
			t.handleConnection(conn, handler)
		}
	}
}
