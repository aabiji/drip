package p2p

import "context"

type Node struct {
	sender   Sender
	receiver Receiver
	finder   PeerFinder
	peers    map[string]*PeerConnection
	ctx      context.Context
	port     int
}

func NewNode(ctx context.Context, downloadFolder *string) *Node {
	n := &Node{
		sender:   NewSender(),
		receiver: NewReceiver(downloadFolder),
		peers:    make(map[string]*PeerConnection),
		port:     getUnusedPort(),
		ctx:      ctx,
	}

	n.finder = NewPeerFinder(n.port, ctx, n.addPeer, n.removePeer)
	go func() {
		if err := n.finder.Run(); err != nil {
			panic(err)
		}
	}()
	return n
}

func (n *Node) SendFiles(recipients []string, files map[string]*File) {
	n.sender.StartTransfer(recipients, files, n.sendMsg)
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
		info.ip, info.id, n.port, info.port,
		n.ctx, n.handlePeerMessage)
	peer.CreateConnection()
	peer.SetupDataChannels()
	n.peers[info.id] = peer
}

func (n *Node) removePeer(peerId string) {
	n.peers[peerId].Close()
	delete(n.peers, peerId)
	n.receiver.Cancel(peerId)
}

func (n *Node) handlePeerMessage(msg Message) {
	switch msg.Type {
	case TRANSFER_RESPONSE:
		response, err := Deserialize[TransferResponse](msg)
		if err != nil {
			panic(err)
		}
		n.sender.HandleTransferResponse(msg.Sender, response, n.sendMsg)
	case TRANSFER_REQUEST:
		id, err := Deserialize[string](msg)
		if err != nil {
			panic(err)
		}
		n.receiver.HandleRequest(id)
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
