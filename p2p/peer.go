package p2p

import (
	"log"
	"net"
	"time"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	Ip        net.IP
	Id        string
	Connected bool

	lastHeardFrom time.Time

	makingOffer bool
	polite      bool
	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel

	packetQueue   chan Message
	udpPort       int
	udpConnection *net.UDPConn

	syncer *FileSyncer
}

func (p *Peer) Close() {
	p.dataChannel.GracefulClose()
	p.udpConnection.Close()
}

func (p *Peer) CreateConnection() {
	var err error
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302"}}},
	}
	p.connection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	p.packetQueue = make(chan Message, 25)

	p.connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnected:
			p.Connected = true
		case webrtc.PeerConnectionStateDisconnected,
			webrtc.PeerConnectionStateFailed,
			webrtc.PeerConnectionStateClosed:
			p.Connected = false
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

		p.packetQueue <- NewMessage(OFFER_PACKET, offer)

		p.makingOffer = false

		log.Println("Sending an offer")
	})

	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		p.packetQueue <- NewMessage(ICE_PACKET, i)
		log.Println("Sending an ice candidate")
	})
}

func (p *Peer) SetupDataChannel() {
	if p.polite { // on the responder's end of the connection
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			log.Println("Accepting a data channel")

			p.dataChannel = dataChannel
			p.dataChannel.OnOpen(p.dataChannelSender)
			p.dataChannel.OnMessage(p.dataChannelReceiver)
			p.dataChannel.OnClose(p.handlePeerDisconnect)
		})
	} else { // on the initiator's end of the connection
		var err error
		log.Println("Creating a data channel")
		p.dataChannel, err = p.connection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}

		p.dataChannel.OnOpen(p.dataChannelSender)
		p.dataChannel.OnMessage(p.dataChannelReceiver)
		p.dataChannel.OnClose(p.handlePeerDisconnect)
	}
}

func (p *Peer) dataChannelSender() {
	// send pending messages over the data channel
	p.Connected = true
	for msg := range p.syncer.PendingMessages[p.Id] {
		log.Printf("Sending: %v\n", msg)
		p.dataChannel.Send(msg.Serialize())
	}
}

func (p *Peer) dataChannelReceiver(channelMsg webrtc.DataChannelMessage) {
	msg := Deserialize(channelMsg.Data)
	if msg.SenderId == getDeviceName() {
		return // ignore our own messages
	}
	log.Printf("Receiving: %v\n", msg)

	response := p.syncer.HandleMessage(msg)
	p.dataChannel.Send(response.Serialize())
}

func (p *Peer) handlePeerDisconnect() {
	log.Println("Peer disconnected")
	// TODO! --> peer disconnected or the data channel closed
	p.Connected = false
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
	p.packetQueue <- NewMessage(ANSWER_PACKET, answer)
	log.Println("Accepting an offer")
}

func (p *Peer) handleAnswer(msg Message) {
	answer, err := DeserializeInto[webrtc.SessionDescription](msg)
	if err != nil {
		panic(err)
	}
	if p.connection.SetRemoteDescription(answer); err != nil {
		panic(err)
	}
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

// FIXME: I know what's happening...we're trying to connect but when we send out our
// udp packets, the peer might not be up, and might miss the packet
// Should we use tcp?? How should we fix/
func (p *Peer) RunClientAndServer(devicePort int) {
	bufferSize := 65536

	// reading from the peer's port
	var err error
	listenAddr := &net.UDPAddr{Port: p.udpPort, IP: net.IPv4zero}
	p.udpConnection, err = net.ListenUDP("udp", listenAddr)
	if err != nil {
		panic(err)
	}

	// Enable broadcast
	p.udpConnection.SetWriteBuffer(bufferSize)
	p.udpConnection.SetReadBuffer(bufferSize)

	// forward json data written to the channel to the client over UDP
	// write to our own port
	go func() {
		broadcastAddr := &net.UDPAddr{Port: devicePort, IP: net.IPv4bcast}
		for pkt := range p.packetQueue {
			p.udpConnection.WriteToUDP(pkt.Serialize(), broadcastAddr)
		}
	}()

	// receive and handle peer signals
	go func() {
		for {
			buffer := make([]byte, bufferSize)
			length, _, err := p.udpConnection.ReadFromUDP(buffer[0:])
			if err != nil {
				panic(err)
			}

			msg := Deserialize(buffer[0:length])
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
	}()
}
