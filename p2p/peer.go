package p2p

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/pion/webrtc/v4"
)

type PeerConnection struct {
	id          string
	makingOffer bool
	polite      bool

	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel
	server      TcpServer

	ctx             context.Context
	cancel          context.CancelFunc
	transferHandler TransferMessageHandler
}

func NewPeer(
	ip net.IP, id string,
	devicePort int, port int,
	parentCtx context.Context,
	handler TransferMessageHandler,
) *PeerConnection {
	ourAddr := fmt.Sprintf(":%d", devicePort)
	peerAddr := fmt.Sprintf("%s:%d", ip.String(), port)
	ctx, cancel := context.WithCancel(parentCtx)

	// This will be used for perfect negotiation. Being polite will
	// mean we forego our own offer when we receive an offer from a peer.
	// Being impolite will mean we ignore the peer's offer and continue with
	// our own. This way, we avoid collisions by knowing that only one peer
	// is able to initiate a connection
	polite := id < getDeviceName()

	return &PeerConnection{
		makingOffer:     false,
		polite:          polite,
		server:          NewTcpServer(ourAddr, peerAddr, ctx),
		ctx:             ctx,
		cancel:          cancel,
		transferHandler: handler,
	}
}

// Will also call the dataChannel's OnClose
func (p *PeerConnection) Close() {
	p.cancel()
	if p.connection != nil {
		p.connection.Close()
		p.connection = nil
	}
	if p.dataChannel != nil {
		p.dataChannel.GracefulClose()
		p.dataChannel = nil
	}
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

		p.server.QueueMessage(NewMessage(OFFER_TCP_PACKET, offer))
		p.makingOffer = false

		log.Println("Sending an offer")
	})

	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		p.server.QueueMessage(NewMessage(ICE_TCP_PACKET, i))
		log.Println("Sending an ice candidate")
	})
}

func (p *PeerConnection) handleReceivingData() {
	p.dataChannel.OnMessage(func(channelMsg webrtc.DataChannelMessage) {
		msg := GetMessage(channelMsg.Data)
		if msg.SenderId == getDeviceName() {
			return // ignore our own messages
		}

		p.transferHandler(msg, func(reply *Message) {
			if reply != nil {
				p.dataChannel.Send(reply.Serialize())
			}
		})
	})
}

func (p *PeerConnection) SetupDataChannels() {
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
	switch msg.MessageType {
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
