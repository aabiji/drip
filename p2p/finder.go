package p2p

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

type PeerFinder struct {
	Peers map[string]*Peer
	mutex sync.Mutex // guards Peers

	transferService *TransferService

	deviceId    string
	devicePort  int
	serviceType string

	peerRemovalTimeout time.Duration
	queryFrequency     time.Duration
	server             *mdns.Server
}

func NewPeerFinder(debugMode bool, t *TransferService) PeerFinder {
	return PeerFinder{
		deviceId:           getDeviceName(debugMode),
		devicePort:         getUnusedPort(),
		Peers:              make(map[string]*Peer),
		serviceType:        "_fileshare._tcp.local.",
		peerRemovalTimeout: time.Second * 15,
		queryFrequency:     time.Second * 10,
		transferService:    t,
	}
}

func (f *PeerFinder) ConnectToPeer(id string) bool {
	peer, ok := f.Peers[id]
	if !ok {
		return false // Not aware of the peer -- TODO: how to handle?
	}
	if peer.ConnectionState == DISCONNECTED {
		peer.CreateConnection()
		peer.RunClientAndServer(f.devicePort)
		peer.SetupDataChannel()
	}
	return true
}

func (f *PeerFinder) broadcastOurService() error {
	hostname := fmt.Sprintf("%s.local.", f.deviceId)

	service, err := mdns.NewMDNSService(
		f.deviceId, f.serviceType, "local.", hostname,
		f.devicePort, []net.IP{getDeviceIP()}, []string{})
	if err != nil {
		return err
	}

	f.server, err = mdns.NewServer(&mdns.Config{Zone: service})
	return err
}

func (f *PeerFinder) addPeer(entry *mdns.ServiceEntry) {
	peerId := strings.Split(entry.Host, ".")[0]
	if peerId == f.deviceId {
		return // ignore ourselves
	}

	// This will be used for perfect negotiation. Being polite will
	// mean we forego our own offer when we receive an offer from a peer.
	// Being impolite will mean we ignore the peer's offer and continue with
	// our own. This way, we avoid collisions by knowing that only one peer
	// is able to initiate a connection
	polite := peerId < f.deviceId

	f.mutex.Lock()

	_, exists := f.Peers[peerId]
	if exists {
		f.Peers[peerId].lastHeardFrom = time.Now()
	} else {
		peer := &Peer{
			Ip:              entry.AddrV4,
			Id:              peerId,
			polite:          polite,
			udpPort:         entry.Port,
			lastHeardFrom:   time.Now(),
			transferService: f.transferService,
		}
		f.Peers[peerId] = peer
		f.transferService.AddPeer(peerId)
	}

	f.mutex.Unlock()
}

// Listen for broadcasts from other devices every 10 seconds
func (f *PeerFinder) listenForBroadcasts(eventChannel chan string, ctx context.Context) error {
	// Start lisening to the broadcasts of other devices
	entriesChannel := make(chan *mdns.ServiceEntry, 25)
	go func() {
		for entry := range entriesChannel {
			f.addPeer(entry)
			eventChannel <- "peers-updated"
		}
	}()

	original := log.Writer()

	for {
		params := mdns.DefaultParams(f.serviceType)
		params.Entries = entriesChannel
		params.Timeout = f.queryFrequency

		log.SetOutput(io.Discard) // disable the logging mdns does
		if err := mdns.Query(params); err != nil {
			return err
		}
		log.SetOutput(original)

		// Remove peers we haven't heard from in a while
		f.mutex.Lock()
		for key, value := range f.Peers {
			if time.Since(value.lastHeardFrom) >= f.peerRemovalTimeout {
				delete(f.Peers, key)
			}
		}
		f.mutex.Unlock()

		// Stop looping when we receive a shutdown signal
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			continue
		}
	}
}

func (f *PeerFinder) Run(eventChannel chan string, ctx context.Context) error {
	if err := f.broadcastOurService(); err != nil {
		return err
	}

	if err := f.listenForBroadcasts(eventChannel, ctx); err != nil {
		return err
	}

	return f.server.Shutdown()
}
