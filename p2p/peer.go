package p2p

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	Id            string
	LastHeardFrom time.Time

	makingOffer bool
	polite      bool

	Webrtc    Medium
	tcpMedium Medium

	ctx    context.Context
	cancel context.CancelFunc

	connection *webrtc.PeerConnection
	downloader *Downloader
	appEvents  chan Message
}

type peerInfo struct {
	ip         net.IP
	id         string
	port       int
	devicePort int
	downloader *Downloader
	appEvents  chan Message
	parentCtx  context.Context
}

func NewPeer(info peerInfo) *Peer {
	ourAddr := fmt.Sprintf(":%d", info.devicePort)
	peerAddr := fmt.Sprintf("%s:%d", info.ip.String(), info.port)
	ctx, cancel := context.WithCancel(info.parentCtx)

	// This will be used for perfect negotiation. Being polite will
	// mean we forego our own offer when we receive an offer from a peer.
	// Being impolite will mean we ignore the peer's offer and continue with
	// our own. This way, we avoid collisions by knowing that only one peer
	// is able to initiate a connection
	polite := info.id < getDeviceName()

	return &Peer{
		Id:            info.id,
		LastHeardFrom: time.Now(),
		makingOffer:   false,
		polite:        polite,
		tcpMedium:     NewTCPMedium(ourAddr, peerAddr),
		ctx:           ctx,
		cancel:        cancel,
		downloader:    info.downloader,
		appEvents:     info.appEvents,
	}
}

// Will also call the dataChannel's OnClose
func (p *Peer) Close() {
	p.cancel()
	if p.connection != nil {
		p.connection.Close()
		p.connection = nil
	}
	if p.Webrtc != nil {
		p.Webrtc.Close()
		p.Webrtc = nil
	}
	p.downloader.CancelSessions(p.Id)
}

// The default value for an interface is nil, so we want to avoid
// panicking when we haven't opened/received a data channel connection yet
func (p *Peer) Connected() bool { return p.Webrtc != nil && p.Webrtc.Connected() }

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
		case webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:
			p.Close()
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

func (p *Peer) SetupDataChannels() {
	handler := func(msg Message) {
		// Forward session responses to the frontend
		if msg.MessageType == SESSSION_RESPONSE {
			p.appEvents <- msg
			return
		}

		reply := p.downloader.ReceiveMessage(msg)
		if reply != nil {
			p.Webrtc.QueueMessage(*reply)
		}
	}

	go func() { p.tcpMedium.ForwardMessages(p.ctx) }()
	go func() { p.tcpMedium.ReceiveMessages(p.ctx, p.handlePeerMessage) }()

	if p.polite {
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			log.Println("Accepting a data channel -- responder")
			p.Webrtc = NewWebRTCMedium(dataChannel)
			p.Webrtc.ForwardMessages(p.ctx)
			p.Webrtc.ReceiveMessages(p.ctx, handler)
		})
	} else {
		log.Println("Creating a data channel -- initiator")
		dataChannel, err := p.connection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}
		p.Webrtc = NewWebRTCMedium(dataChannel)
		p.Webrtc.ForwardMessages(p.ctx)
		p.Webrtc.ReceiveMessages(p.ctx, handler)
	}
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
