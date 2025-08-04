package p2p

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

type Message struct {
	Sender     string   `json:",omitempty"`
	Recipients []string `json:",omitempty"`
	Type       int
	Data       []byte
}

func NewMessage[T any](messageType int, value T) Message {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return Message{
		Type:   messageType,
		Data:   encoded,
		Sender: deviceName(),
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

func deviceIP() net.IP {
	// get the prefered outbound ip address of this device
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err) // getting the ip is a must
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

func deviceName() string {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s-%d", name, os.Getpid())
}

func getUnusedPort() int {
	// get the os to give a random free port
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

func fallocate(file *os.File, offset int64, length int64) error {
	if length == 0 {
		return nil
	}
	return unix.Fallocate(int(file.Fd()), 0, offset, length)
}
