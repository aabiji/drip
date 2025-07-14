package p2p

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

type PeerFinder struct {
	Peers map[string]*Peer
	mutex sync.Mutex // guards Peers

	syncer *FileSyncer

	devicePort  int
	serviceType string

	peerRemovalTimeout time.Duration
	queryFrequency     time.Duration
	server             *mdns.Server
}

func NewPeerFinder(debugMode bool, t *FileSyncer) PeerFinder {
	return PeerFinder{
		devicePort:         getUnusedPort(),
		Peers:              make(map[string]*Peer),
		serviceType:        "_fileshare._tcp.local.",
		peerRemovalTimeout: time.Second * 15,
		queryFrequency:     time.Second * 10,
		syncer:             t,
	}
}

func (f *PeerFinder) broadcastOurService() error {
	id := getDeviceName()
	hostname := fmt.Sprintf("%s.local.", id)

	service, err := mdns.NewMDNSService(
		id, f.serviceType, "local.", hostname,
		f.devicePort, []net.IP{getDeviceIP()}, []string{})
	if err != nil {
		return err
	}

	f.server, err = mdns.NewServer(&mdns.Config{Zone: service})
	return err
}

func (f *PeerFinder) addPeer(entry *mdns.ServiceEntry) {
	peerId := strings.Split(entry.Host, ".")[0]
	if peerId == getDeviceName() {
		return // ignore ourselves
	}

	// This will be used for perfect negotiation. Being polite will
	// mean we forego our own offer when we receive an offer from a peer.
	// Being impolite will mean we ignore the peer's offer and continue with
	// our own. This way, we avoid collisions by knowing that only one peer
	// is able to initiate a connection
	polite := peerId < getDeviceName()

	f.mutex.Lock()

	_, exists := f.Peers[peerId]
	if exists {
		f.Peers[peerId].LastHeardFrom = time.Now()
	} else {
		peer := NewPeer(entry.AddrV4, peerId, polite, f.syncer, entry.Port, f.devicePort)
		peer.CreateConnection()
		peer.SetupDataChannels()
		f.Peers[peerId] = peer
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

	for {
		params := mdns.DefaultParams(f.serviceType)
		params.Entries = entriesChannel
		params.Timeout = f.queryFrequency

		if err := mdns.Query(params); err != nil {
			return err
		}

		// Remove peers we haven't heard from in a while
		f.mutex.Lock()
		for key, value := range f.Peers {
			if time.Since(value.LastHeardFrom) >= f.peerRemovalTimeout {
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
