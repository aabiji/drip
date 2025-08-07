package p2p

import (
	"context"
)

// event types used to communicate between the peer to peer node and the frontend
// TRANSFER_REQUEST, TRANSFER_RESPONSE and TRANSFER_CANCELLED are also used
const (
	ADDED_PEER = iota + 200
	REMOVED_PEER
	NOTIFY_COMPLETION
	SEND_FILES
	AUTH_GRANTED
	TRANSFER_REJECTED
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

func (n *Node) GetProgressReport(transferId string) ProgressReport {
	return n.sender.GetProgressReport(transferId)
}

func (n *Node) CancelTransfer(transferId string) {
	n.sender.CancelTransfer(transferId, n.sendMsg)
}

func (n *Node) Shutdown() {
	n.receiver.Close()
	for _, peer := range n.peers {
		peer.Close()
	}
}

func (n *Node) sendMsg(msg Message) {
	for _, peer := range msg.Recipients {
		if msg.Type == TRANSFER_CHUNK {
			n.peers[peer].pendingChunks <- msg
		} else {
			n.peers[peer].pendingMesages <- msg
		}
	}
}

func (n *Node) addPeer(info PeerInfo) {
	peer := NewPeer(
		info.Ip, info.Id, n.port, info.Port,
		n.ctx, n.nodeEvents, n.handlePeerMessage)
	peer.CreateConnection()
	peer.SetupChannels()
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
		response, err := Deserialize[TransferResponse](msg)
		if err != nil {
			panic(err)
		}
		if !response.Authorized {
			n.appEvents <- NewMessage(TRANSFER_REJECTED, "")
		}
		n.sender.HandleTransferResponse(msg.Sender, response, n.sendMsg)
	case TRANSFER_INFO:
		info, err := Deserialize[Transfer](msg)
		if err != nil {
			panic(err)
		}
		n.receiver.HandleInfo(info)
	case TRANSFER_CANCELLED:
		id, err := Deserialize[string](msg)
		if err != nil {
			panic(err)
		}
		n.receiver.HandleCancel(id)
	case TRANSFER_CHUNK:
		chunk, err := Deserialize[Chunk](msg)
		if err != nil {
			panic(err)
		}
		go n.receiver.HandleChunk(chunk)
	}
}
