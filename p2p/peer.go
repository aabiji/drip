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
	makingOffer   bool
	polite        bool

	ctx    context.Context
	cancel context.CancelFunc

	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel
	server      TcpServer

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
		server:        NewTcpServer(ourAddr, peerAddr, ctx),
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
	if p.dataChannel != nil {
		p.dataChannel.GracefulClose()
		p.dataChannel = nil
	}
	p.downloader.CancelSessions(p.Id)
}

func (p *Peer) Connected() bool { return p.dataChannel != nil }

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

		p.server.QueueMessage(NewMessage(OFFER_PACKET, offer))
		p.makingOffer = false

		log.Println("Sending an offer")
	})

	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		p.server.QueueMessage(NewMessage(ICE_PACKET, i))
		log.Println("Sending an ice candidate")
	})
}

func (p *Peer) handleReceivingData() {
	p.dataChannel.OnMessage(func(channelMsg webrtc.DataChannelMessage) {
		msg := GetMessage(channelMsg.Data)
		if msg.SenderId == getDeviceName() {
			return // ignore our own messages
		}

		if msg.MessageType == SESSSION_RESPONSE {
			p.appEvents <- msg
			return
		}

		reply := p.downloader.ReceiveMessage(msg)
		if reply != nil {
			p.dataChannel.Send(reply.Serialize())
		}
	})
}

func (p *Peer) SetupDataChannels() {
	go func() { p.server.ForwardMessages() }()
	go func() { p.server.ReceiveMessages(p.handlePeerMessage) }()

	if p.polite {
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			p.dataChannel = dataChannel
			p.handleReceivingData()
			log.Println("Accepting a data channel -- responder")
		})
	} else {
		var err error
		p.dataChannel, err = p.connection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}
		p.handleReceivingData()
		log.Println("Creating a data channel -- initiator")
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
	p.server.QueueMessage(NewMessage(ANSWER_PACKET, answer))
	log.Println("Accepting an offer")
}

func (p *Peer) handlePeerMessage(msg Message) {
	switch msg.MessageType {
	case ANSWER_PACKET:
		answer, err := Deserialize[webrtc.SessionDescription](msg)
		if err != nil {
			panic(err)
		}
		p.connection.SetRemoteDescription(answer)
		log.Println("Accepting an answer")

	case ICE_PACKET:
		candidate, err := Deserialize[webrtc.ICECandidate](msg)
		if err != nil {
			panic(err)
		}
		p.connection.AddICECandidate(candidate.ToJSON())
		log.Println("Adding an ICE candidate")

	case OFFER_PACKET:
		p.handleOffer(msg)

	default:
		panic("uknown signal type")
	}
}
