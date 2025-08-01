package p2p

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
)

type Node struct {
	downloader Downloader
	sender     Sender
	finder     PeerFinder
	peers      map[string]*PeerConnection

	ctx    context.Context
	cancel context.CancelFunc
	events chan Message
	port   int
}

// TODO: shutdown function
func NewNode(
	downloadFolder *string,
	authorize AuthorizeCallback,
	notify NotifyCallback,
) *Node {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan Message)
	port := getUnusedPort()

	n := &Node{
		downloader: NewDownloader(downloadFolder, authorize, notify),
		peers:      make(map[string]*PeerConnection),
		port:       port,
		ctx:        ctx,
		cancel:     cancel,
		events:     events,
	}
	n.finder = NewPeerFinder(port, ctx, n.addPeer, n.removePeer)
	return n
}

func (n *Node) handleTransferMessage(msg Message, handler ReplyHandler) {
	if msg.MessageType == TRANSFER_SESSION_AUTH {
		n.sender.handlePeerResponse(msg)
		handler(nil)
	} else {
		reply := n.downloader.ReceiveMessage(msg)
		handler(reply)
	}
}

func (n *Node) addPeer(info PeerInfo) {
	peer := NewPeer(
		info.ip, info.id, n.port, info.port,
		n.ctx, n.handleTransferMessage)
	peer.CreateConnection()
	peer.SetupDataChannels()
	n.peers[info.id] = peer
}

func (n *Node) removePeer(peerId string) {
	n.peers[peerId].Close()
	delete(n.peers, peerId)
	n.downloader.CancelSessions(peerId)
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

var DEBUG_MODE bool = true
var cached_device_name string = ""

func getDeviceName() string {
	if len(cached_device_name) > 0 {
		return cached_device_name
	}

	if DEBUG_MODE {
		id := rand.Intn(1000000)
		cached_device_name = fmt.Sprintf("peer-%d", id)
		return cached_device_name
	}

	name, err := os.Hostname()
	if err != nil {
		panic(err) // getting the hostname is a must
	}
	cached_device_name = name
	return name
}
