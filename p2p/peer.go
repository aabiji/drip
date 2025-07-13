package p2p

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/pion/webrtc/v4"
)

const ( // connection states
	DISCONNECTED = 0
	CONNECTED    = 1
	CONNECTING   = 2
)

const ( // packet types
	OFFER_PACKET  = 0
	ANSWER_PACKET = 1
	ICE_PACKET    = 2
)

// TODO: this is doing way too much stuff...refactor
type Peer struct {
	Ip              net.IP
	Id              string
	ConnectionState int

	lastHeardFrom time.Time

	makingOffer bool
	polite      bool
	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel

	packetQueue   chan Message
	udpPort       int
	udpConnection *net.UDPConn

	transferService *TransferService
}

func (p *Peer) Close() {
	p.dataChannel.GracefulClose()
	p.udpConnection.Close()
}

func (p *Peer) CreateConnection() {
	var err error
	config := webrtc.Configuration{ICEServers: []webrtc.ICEServer{}}
	p.connection, err = webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	p.packetQueue = make(chan Message, 25)

	p.connection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		switch state {
		case webrtc.PeerConnectionStateConnecting:
			p.ConnectionState = CONNECTING
		case webrtc.PeerConnectionStateConnected:
			p.ConnectionState = CONNECTED
		default:
			p.ConnectionState = DISCONNECTED
		}

		if p.ConnectionState == DISCONNECTED {
			// TODO
			fmt.Println("handling peer disconnect...")
		}
	})

	// Our chance to send an offer
	p.connection.OnNegotiationNeeded(func() {
		p.makingOffer = true

		offer, err := p.connection.CreateOffer(nil)
		if err != nil {
			panic(err)
		}
		p.connection.SetLocalDescription(offer)

		p.packetQueue <- NewMessage(OFFER_PACKET, offer)
		p.makingOffer = false
	})

	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		p.packetQueue <- NewMessage(ICE_PACKET, i)
	})
}

func (p *Peer) dataChannelSender() {
	// send pending messages over the data channel
	for msg := range p.transferService.Pending[p.Id] {
		p.dataChannel.Send(msg.Serialize())
	}
}

func (p *Peer) dataChannelReceiver(channelMsg webrtc.DataChannelMessage) {
	msg := Message{}
	err := json.Unmarshal(channelMsg.Data, &msg)
	if err != nil {
		panic(err)
	}
	response := p.transferService.HandleMessage(msg)
	p.dataChannel.Send(response.Serialize())
}

func (p *Peer) dataChannelClose() {
	fmt.Println("data channel closed...handle cleanup")
}

func (p *Peer) SetupDataChannel() {
	if p.polite { // on the responder's end of the connection
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			p.dataChannel = dataChannel
			p.dataChannel.OnOpen(p.dataChannelSender)
			p.dataChannel.OnMessage(p.dataChannelReceiver)
			p.dataChannel.OnClose(p.dataChannelClose)
		})
	} else { // on the initiator's end of the connection
		var err error
		p.dataChannel, err = p.connection.CreateDataChannel("data", nil)
		if err != nil {
			panic(err)
		}
		p.dataChannel.OnOpen(p.dataChannelSender)
		p.dataChannel.OnMessage(p.dataChannelReceiver)
		p.dataChannel.OnClose(p.dataChannelClose)
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

	var offer webrtc.SessionDescription
	err := json.Unmarshal(msg.Data, &offer)
	if err != nil {
		panic(err)
	}
	p.connection.SetRemoteDescription(offer)

	answer, err := p.connection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	p.connection.SetLocalDescription(answer)
	p.packetQueue <- NewMessage(ANSWER_PACKET, answer)
}

func (p *Peer) handleAnswer(msg Message) {
	var answer webrtc.SessionDescription
	err := json.Unmarshal(msg.Data, &answer)
	if err != nil {
		panic(err)
	}
	p.connection.SetRemoteDescription(answer)
}

func (p *Peer) handleICECandidate(msg Message) {
	var candidate webrtc.ICECandidate
	err := json.Unmarshal(msg.Data, &candidate)
	if err != nil {
		panic(err)
	}
	p.connection.AddICECandidate(candidate.ToJSON())
}

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

			var msg Message
			err = json.Unmarshal(buffer[0:length], &msg)
			if err != nil {
				panic(err)
			}
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
