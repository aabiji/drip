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

type packet struct {
	PacketType int
	Data       []byte
	// TODO: use these!
	Recipient string
	Sender    string
}

type Peer struct {
	Ip              net.IP
	Id              string
	IsMobile        bool
	ConnectionState int

	lastHeardFrom time.Time

	makingOffer bool
	connection  *webrtc.PeerConnection
	dataChannel *webrtc.DataChannel

	packetChannel chan packet
	udpPort       int
	udpConnection *net.UDPConn
}

// This will be used for perfect negotiation. Being polite will
// mean we forego our own offer when we receive an offer from a peer.
// Being impolite will mean we ignore the peer's offer and continue with
// our own. This way, we avoid collisions by knowing that only one peer
// is able to initiate a connection
func (p *Peer) isPolite() bool { return p.Id < DEVICE_ID }

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
	p.packetChannel = make(chan packet, 25)

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

		jsonOffer, err := json.Marshal(offer)
		if err != nil {
			panic(err)
		}

		p.packetChannel <- packet{
			PacketType: OFFER_PACKET, Data: jsonOffer,
			Sender: DEVICE_ID, Recipient: p.Id,
		}

		p.makingOffer = false
	})

	p.connection.OnICECandidate(func(i *webrtc.ICECandidate) {
		jsonCandidate, err := json.Marshal(i)
		if err != nil {
			panic(err)
		}

		p.packetChannel <- packet{
			PacketType: ICE_PACKET, Data: jsonCandidate,
			Sender: DEVICE_ID, Recipient: p.Id,
		}
	})
}

func (p *Peer) SetupDataChannel() {
	if p.isPolite() { // on the responder's end of the connection
		p.connection.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
			dataChannel.OnOpen(func() {
				fmt.Println("recipient: data channel opened, can start sending data...")
				data := []byte("hello :)")
				dataChannel.Send(data)
			})

			dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
				fmt.Println("received data: ", msg.Data)
			})

			//err := dataChannel.Close()
			dataChannel.OnClose(func() {
				fmt.Println("data channel closed...handle cleanup")
			})
		})
		return
	}

	// on the initiator's end of the connection
	var err error
	p.dataChannel, err = p.connection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	p.dataChannel.OnOpen(func() {
		fmt.Println("sender: data channel opened, can start sending data...")
		data := []byte("hello :)")
		p.dataChannel.Send(data)
	})

	p.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Println("received data: ", msg.Data)
	})

	//err := dataChannel.Close()
	p.dataChannel.OnClose(func() {
		fmt.Println("data channel closed...handle cleanup")
	})
}

func (p *Peer) handleOffer(pkt packet) {
	// are we getting an offer in the middle of sending ours?
	negotiating := p.connection.SignalingState() != webrtc.SignalingStateStable
	offerCollision := negotiating || p.makingOffer

	if offerCollision && !p.isPolite() {
		return // Ignore the peer's offer and so we can move forward with our own
	}
	p.makingOffer = false

	var offer webrtc.SessionDescription
	err := json.Unmarshal(pkt.Data, &offer)
	if err != nil {
		panic(err)
	}
	p.connection.SetRemoteDescription(offer)

	answer, err := p.connection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}
	p.connection.SetLocalDescription(answer)

	jsonAnswer, err := json.Marshal(answer)
	if err != nil {
		panic(err)
	}
	p.packetChannel <- packet{
		PacketType: ANSWER_PACKET, Data: jsonAnswer,
		Sender: DEVICE_ID, Recipient: p.Id,
	}
}

func (p *Peer) handleAnswer(pkt packet) {
	var answer webrtc.SessionDescription
	err := json.Unmarshal(pkt.Data, &answer)
	if err != nil {
		panic(err)
	}
	p.connection.SetRemoteDescription(answer)
}

func (p *Peer) handleICECandidate(pkt packet) {
	var candidate webrtc.ICECandidate
	err := json.Unmarshal(pkt.Data, &candidate)
	if err != nil {
		panic(err)
	}
	p.connection.AddICECandidate(candidate.ToJSON())
}

func (p *Peer) RunClientAndServer() {
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
		broadcastAddr := &net.UDPAddr{Port: DEVICE_PORT, IP: net.IPv4bcast}
		for pkt := range p.packetChannel {
			jsonPacket, err := json.Marshal(pkt)
			if err != nil {
				panic(err)
			}
			p.udpConnection.WriteToUDP(jsonPacket, broadcastAddr)
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

			var pkt packet
			err = json.Unmarshal(buffer[0:length], &pkt)
			if err != nil {
				panic(err)
			}
			switch pkt.PacketType {
			case OFFER_PACKET:
				p.handleOffer(pkt)
			case ANSWER_PACKET:
				p.handleAnswer(pkt)
			case ICE_PACKET:
				p.handleICECandidate(pkt)
			default:
				panic("uknown signal type")
			}
		}
	}()
}
