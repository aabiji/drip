package p2p

import (
	"context"
	"fmt"
	"github.com/pion/webrtc/v4"
	"log"
	"net"
	"time"
)

type Peer struct {
	Id            string
	LastHeardFrom time.Time

	makingOffer bool
	connected   bool
	polite      bool

	Webrtc    Medium
	tcpMedium Medium

	connection *webrtc.PeerConnection
	syncer     *FileSyncer
	appEvents  chan Message
}

func NewPeer(
	ip net.IP, id string, port int, devicePort int,
	syncer *FileSyncer, appEvents chan Message) *Peer {
	ourAddr := fmt.Sprintf(":%d", devicePort)
	peerAddr := fmt.Sprintf("%s:%d", ip.String(), port)
	packets := make(chan Message, 25)

	// This will be used for perfect negotiation. Being polite will
	// mean we forego our own offer when we receive an offer from a peer.
	// Being impolite will mean we ignore the peer's offer and continue with
	// our own. This way, we avoid collisions by knowing that only one peer
	// is able to initiate a connection
	polite := id < getDeviceName()

	return &Peer{
		Id:            id,
		polite:        polite,
		LastHeardFrom: time.Now(),
		syncer:        syncer,
		tcpMedium:     &TCPMedium{packets, peerAddr, ourAddr},
		appEvents:     appEvents,
	}
}

func (p *Peer) Close() { p.connection.Close() }

func (p *Peer) Connected() bool { return p.Webrtc.Connected() || p.connected }

func (p *Peer) CreateConnection() {
	var err error
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}
	p.connection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	p.connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			p.connected = true
		case webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:
			p.connected = false
			p.handlePeerDisconnect()
		default:
			// do nothing
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

		p.tcpMedium.QueueMessage(NewMessage(OFFER_PACKET, offer))
		p.makingOffer = false

		log.Println("Sending an offer")
	})

	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		p.tcpMedium.QueueMessage(NewMessage(ICE_PACKET, i))
		log.Println("Sending an ice candidate")
	})
}

func (p *Peer) SetupDataChannels(ctx context.Context) {
	handler := func(msg Message) {
		// Forward transfer replys to the frontend
		if msg.MessageType == TRANSFER_REPLY {
			p.appEvents <- msg
		}

		reply := p.syncer.HandleMessage(msg)
		if reply != nil {
			p.Webrtc.QueueMessage(*reply)
		}
	}

	go func() { p.tcpMedium.ForwardMessages(ctx) }()
	go func() { p.tcpMedium.ReceiveMessages(ctx, p.handlePeerMessage) }()

	if p.polite {
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			log.Println("Accepting a data channel -- responder")
			p.Webrtc = NewWebRTCMedium(dataChannel)
			p.Webrtc.ForwardMessages(ctx)
			p.Webrtc.ReceiveMessages(ctx, handler)
		})
	} else {
		log.Println("Creating a data channel -- initiator")
		dataChannel, err := p.connection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}
		p.Webrtc = NewWebRTCMedium(dataChannel)
		p.Webrtc.ForwardMessages(ctx)
		p.Webrtc.ReceiveMessages(ctx, handler)
	}
}

func (p *Peer) handlePeerDisconnect() {
	log.Println("Peer disconnected")
	// TODO! --> peer disconnected or the data channel closed
	p.connected = false
}

func (p *Peer) handleOffer(msg Message) {
	// are we getting an offer in the middle of sending ours?
	negotiating := p.connection.SignalingState() != webrtc.SignalingStateStable
	offerCollision := negotiating || p.makingOffer

	if offerCollision && !p.polite {
		return // Ignore the peer's offer and so we can move forward with our own
	}
	p.makingOffer = false

	offer, err := DeserializeInto[webrtc.SessionDescription](msg)
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
	p.tcpMedium.QueueMessage(NewMessage(ANSWER_PACKET, answer))
	log.Println("Accepting an offer")
}

func (p *Peer) handleAnswer(msg Message) {
	answer, err := DeserializeInto[webrtc.SessionDescription](msg)
	if err != nil {
		panic(err)
	}
	p.connection.SetRemoteDescription(answer)
	log.Println("Accepting an answer")
}

func (p *Peer) handleICECandidate(msg Message) {
	candidate, err := DeserializeInto[webrtc.ICECandidate](msg)
	if err != nil {
		panic(err)
	}
	p.connection.AddICECandidate(candidate.ToJSON())
	log.Println("Adding an ICE candidate")
}

func (p *Peer) handlePeerMessage(msg Message) {
	switch msg.MessageType {
	case OFFER_PACKET:
		p.handleOffer(msg)
	case ANSWER_PACKET:
		p.handleAnswer(msg)
	case ICE_PACKET:
		p.handleICECandidate(msg)
	default:
		panic("uknown signal type")
	}
}
