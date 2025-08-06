package p2p

import (
	"context"
)

// event types
// TRANSFER_REQUEST and TRANSFER_RESPONSE are also used
const (
	ADDED_PEER = iota + 100
	REMOVED_PEER
	NOTIFY_COMPLETION

	// used by the app
	SEND_FILES
	AUTH_GRANTED
)

type Node struct {
	sender   Sender
	receiver Receiver
	finder   PeerFinder
	peers    map[string]*PeerConnection

	appEvents  chan Message
	nodeEvents chan Message

	ctx  context.Context
	port int
}

func NewNode(
	ctx context.Context, downloadFolder *string,
	appEvents chan Message, nodeEvents chan Message,
) *Node {
	n := &Node{
		sender:     NewSender(),
		receiver:   NewReceiver(downloadFolder, appEvents),
		peers:      make(map[string]*PeerConnection),
		appEvents:  appEvents,
		nodeEvents: nodeEvents,
		port:       getUnusedPort(),
		ctx:        ctx,
	}

	go n.handleNodeEvents()

	// find peers
	n.finder = NewPeerFinder(n.port, ctx, n.nodeEvents)
	go func() {
		if err := n.finder.Run(); err != nil {
			panic(err)
		}
	}()
	return n
}

func (n *Node) SendFiles(recipients []string, files map[string]*File) string {
	return n.sender.StartTransfer(recipients, files, n.sendMsg)
}

func (n *Node) GetFilePercentages(transferId string) map[string]float32 {
	return n.sender.GetFilePercentages(transferId)
}

func (n *Node) Shutdown() {
	n.receiver.Close()
	for _, peer := range n.peers {
		peer.Close()
	}
}

func (n *Node) sendMsg(msg Message) {
	for _, peer := range msg.Recipients {
		n.peers[peer].PendingSending <- msg
	}
}

func (n *Node) addPeer(info PeerInfo) {
	peer := NewPeer(
		info.Ip, info.Id, n.port, info.Port,
		n.ctx, n.nodeEvents, n.handlePeerMessage)
	peer.CreateConnection()
	peer.SetupDataChannels()
	n.peers[info.Id] = peer
	n.appEvents <- NewMessage(ADDED_PEER, info.Id)
}

func (n *Node) handleNodeEvents() {
	for event := range n.nodeEvents {
		switch event.Type {
		case TRANSFER_RESPONSE:
			n.sendMsg(event) // send the response to the sender

		case ADDED_PEER:
			info, err := Deserialize[PeerInfo](event)
			if err != nil {
				panic(err)
			}
			n.addPeer(info)

		case REMOVED_PEER:
			peerId, err := Deserialize[string](event)
			if err != nil {
				panic(err)
			}
			// remove peer
			n.receiver.Cancel(peerId)
			delete(n.peers, peerId)
			n.appEvents <- NewMessage(REMOVED_PEER, peerId)
		}
	}
}

func (n *Node) handlePeerMessage(msg Message) {
	switch msg.Type {
	case TRANSFER_REQUEST:
		n.appEvents <- msg // forward this to the frontend
	case TRANSFER_RESPONSE:
		// TODO: handle the case where we're not authorized
		response, err := Deserialize[TransferResponse](msg)
		if err != nil {
			panic(err)
		}
		n.sender.HandleTransferResponse(msg.Sender, response, n.sendMsg)
	case TRANSFER_INFO:
		info, err := Deserialize[Transfer](msg)
		if err != nil {
			panic(err)
		}
		n.receiver.HandleInfo(info)
	case TRANSFER_CHUNK:
		chunk, err := Deserialize[Chunk](msg)
		if err != nil {
			panic(err)
		}
		n.receiver.HandleChunk(chunk)
	case TRANSFER_CANCELLED:
		id, err := Deserialize[string](msg)
		if err != nil {
			panic(err)
		}
		n.receiver.HandleCancel(id)
	}
}
