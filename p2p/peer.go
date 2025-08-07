package p2p

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

type PeerConnection struct {
	makingOffer bool
	polite      bool
	id          string

	server     TcpServer
	connection *webrtc.PeerConnection
	msgHandler func(Message) // handle messages received from the data channel
	nodeEvents chan Message  // communicate with the peer to peer node

	pendingMesages chan Message
	msgChannel     *webrtc.DataChannel
	pendingChunks  chan Message
	chunksChannel  *webrtc.DataChannel

	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
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
		polite:         polite,
		id:             id,
		server:         NewTcpServer(ourAddr, peerAddr, ctx),
		pendingMesages: make(chan Message, 100),
		pendingChunks:  make(chan Message, 100),
		ctx:            ctx,
		cancel:         cancel,
		msgHandler:     handler,
		nodeEvents:     nodeEvents,
	}
}

// Will also call the dataChannel's OnClose
func (p *PeerConnection) Close() {
	p.closeOnce.Do(func() {
		p.cancel()
		close(p.pendingChunks)
		close(p.pendingMesages)
		p.connection.Close()
		p.chunksChannel.GracefulClose()
		p.msgChannel.GracefulClose()
		p.nodeEvents <- NewMessage(REMOVED_PEER, p.id)
	})
}

func (p *PeerConnection) Connected() bool {
	return p.msgChannel != nil && p.chunksChannel != nil
}

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

func (p *PeerConnection) SetupChannels() {
	go func() { p.server.ForwardMessages() }()
	go func() { p.server.ReceiveMessages(p.handlePeerMessage) }()

	receiveHandler := func(dataChannel *webrtc.DataChannel) {
		dataChannel.OnMessage(func(channelMsg webrtc.DataChannelMessage) {
			msg := GetMessage(channelMsg.Data)
			if msg.Sender != deviceName() {
				p.msgHandler(msg)
			}
		})
	}

	sendHandler := func(dataChannel *webrtc.DataChannel, channel chan Message) {
		bufferSizeLimit := uint64(8 * 1024 * 1024) // 8 megabytes
		for msg := range channel {
			// don't keep too much data buffered in order
			// to reduce congestion and limit memory usage
			for dataChannel.BufferedAmount() > bufferSizeLimit {
				time.Sleep(10 * time.Millisecond)
			}
			dataChannel.Send(msg.Serialize())
		}
	}

	if p.polite {
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			if dataChannel.Label() == "message" {
				p.msgChannel = dataChannel
				receiveHandler(p.msgChannel)
				go sendHandler(p.msgChannel, p.pendingMesages)
			} else {
				p.chunksChannel = dataChannel
				receiveHandler(p.chunksChannel)
				go sendHandler(p.chunksChannel, p.pendingChunks)
			}
			log.Printf("Accepting %s data channel\n", dataChannel.Label())
		})
	} else {
		var err error

		p.msgChannel, err = p.connection.CreateDataChannel("message", nil)
		if err != nil {
			panic(err)
		}
		receiveHandler(p.msgChannel)
		go sendHandler(p.msgChannel, p.pendingMesages)

		p.chunksChannel, err = p.connection.CreateDataChannel("chunk", nil)
		if err != nil {
			panic(err)
		}
		receiveHandler(p.chunksChannel)
		go sendHandler(p.chunksChannel, p.pendingChunks)

		log.Println("Created control and message data channels")
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
