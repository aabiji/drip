package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/pion/webrtc/v4"
)

type PeerConnection struct {
	makingOffer bool
	polite      bool
	id          string
	closed      bool

	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel
	server      TcpServer

	// queue for messages to be sent over the data channel
	PendingSending chan Message
	// handles the messages received from the data channel
	msgHandler func(Message)
	// to communicate with the peer to peer node
	nodeEvents chan Message

	ctx    context.Context
	cancel context.CancelFunc
}

func NewPeer(
	ip net.IP, id string, devicePort int, port int,
	parentCtx context.Context, nodeEvents chan Message,
	handler func(Message),
) *PeerConnection {
	ourAddr := fmt.Sprintf(":%d", devicePort)
	peerAddr := fmt.Sprintf("%s:%d", ip.String(), port)
	ctx, cancel := context.WithCancel(parentCtx)

	// This will be used for perfect negotiation. Being polite will
	// mean we forego our own offer when we receive an offer from a peer.
	// Being impolite will mean we ignore the peer's offer and continue with
	// our own. This way, we avoid collisions by knowing that only one peer
	// is able to initiate a connection
	polite := id < deviceName()

	return &PeerConnection{
		makingOffer:    false,
		closed:         false,
		polite:         polite,
		id:             id,
		server:         NewTcpServer(ourAddr, peerAddr, ctx),
		PendingSending: make(chan Message, 100),
		ctx:            ctx,
		cancel:         cancel,
		msgHandler:     handler,
		nodeEvents:     nodeEvents,
	}
}

// Will also call the dataChannel's OnClose
func (p *PeerConnection) Close() {
	if p.closed {
		return
	}

	p.cancel()
	close(p.PendingSending)
	p.connection.Close()
	p.dataChannel.GracefulClose()
	p.nodeEvents <- NewMessage(REMOVED_PEER, p.id)
	p.closed = true
}

func (p *PeerConnection) Connected() bool { return p.dataChannel != nil }

func (p *PeerConnection) CreateConnection() {
	var err error
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}
	p.connection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	p.connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed {
			p.Close()
		}
	})

	// Our chance to send an offer
	p.connection.OnNegotiationNeeded(func() {
		p.makingOffer = true
		offer, err := p.connection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}
		if err := p.connection.SetLocalDescription(offer); err != nil {
			panic(err)
		}

		p.server.QueueMessage(NewMessage(OFFER_TCP_PACKET, offer))
		p.makingOffer = false

		log.Println("Sending an offer")
	})

	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		p.server.QueueMessage(NewMessage(ICE_TCP_PACKET, i))
		log.Println("Sending an ice candidate")
	})
}

func (p *PeerConnection) SetupDataChannels() {
	go func() { p.server.ForwardMessages() }()
	go func() { p.server.ReceiveMessages(p.handlePeerMessage) }()

	receiveHandler := func() {
		p.dataChannel.OnMessage(func(channelMsg webrtc.DataChannelMessage) {
			msg := GetMessage(channelMsg.Data)
			if msg.Sender != deviceName() {
				p.msgHandler(msg)
			}
		})
	}

	sendHandler := func() {
		for msg := range p.PendingSending {
			p.dataChannel.Send(msg.Serialize())
		}
	}

	if p.polite {
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			p.dataChannel = dataChannel
			receiveHandler()
			go sendHandler()
			log.Println("Accepting a data channel -- responder")
		})
	} else {
		var err error
		p.dataChannel, err = p.connection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}
		receiveHandler()
		go sendHandler()
		log.Println("Creating a data channel -- initiator")
	}
}

func (p *PeerConnection) handleOffer(msg Message) {
	// are we getting an offer in the middle of sending ours?
	negotiating := p.connection.SignalingState() != webrtc.SignalingStateStable
	offerCollision := negotiating || p.makingOffer

	if offerCollision && !p.polite {
		return // Ignore the peer's offer and so we can move forward with our own
	}
	p.makingOffer = false

	offer, err := Deserialize[webrtc.SessionDescription](msg)
	if err != nil {
		panic(err)
	}
	if err := p.connection.SetRemoteDescription(offer); err != nil {
		panic(err)
	}

	answer, err := p.connection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	if err := p.connection.SetLocalDescription(answer); err != nil {
		panic(err)
	}
	p.server.QueueMessage(NewMessage(ANSWER_TCP_PACKET, answer))
	log.Println("Accepting an offer")
}

func (p *PeerConnection) handlePeerMessage(msg Message) {
	switch msg.Type {
	case ANSWER_TCP_PACKET:
		answer, err := Deserialize[webrtc.SessionDescription](msg)
		if err != nil {
			panic(err)
		}
		p.connection.SetRemoteDescription(answer)
		log.Println("Accepting an answer")

	case ICE_TCP_PACKET:
		candidate, err := Deserialize[webrtc.ICECandidate](msg)
		if err != nil {
			panic(err)
		}
		p.connection.AddICECandidate(candidate.ToJSON())
		log.Println("Adding an ICE candidate")

	case OFFER_TCP_PACKET:
		p.handleOffer(msg)

	default:
		panic("uknown signal type")
	}
}
