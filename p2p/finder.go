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
	Peers     map[string]*Peer
	mutex     sync.Mutex // guards Peers
	appEvents chan Message

	devicePort  int
	serviceType string

	peerRemovalTimeout time.Duration
	queryFrequency     time.Duration
	server             *mdns.Server
}

func NewPeerFinder(debugMode bool, appEvents chan Message) PeerFinder {
	return PeerFinder{
		devicePort:         getUnusedPort(),
		Peers:              make(map[string]*Peer),
		serviceType:        "_fileshare._tcp.local.",
		peerRemovalTimeout: time.Second * 15,
		queryFrequency:     time.Second * 10,
		appEvents:          appEvents,
	}
}

func (f *PeerFinder) GetConnectedPeers() []string {
	f.mutex.Lock()
	var ids []string
	for id, peer := range f.Peers {
		if peer.Connected() {
			ids = append(ids, id)
		}
	}
	f.mutex.Unlock()
	return ids
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

func (f *PeerFinder) addPeer(ctx context.Context, entry *mdns.ServiceEntry, d *Downloader) {
	peerId := strings.Split(entry.Host, ".")[0]
	if peerId == getDeviceName() {
		return // ignore ourselves
	}

	f.mutex.Lock()

	_, exists := f.Peers[peerId]
	if exists {
		f.Peers[peerId].LastHeardFrom = time.Now()
	} else {
		peer := NewPeer(peerInfo{
			ip:         entry.AddrV4,
			id:         peerId,
			port:       entry.Port,
			devicePort: f.devicePort,
			downloader: d,
			appEvents:  f.appEvents,
			parentCtx:  ctx,
		})
		peer.CreateConnection()
		peer.SetupDataChannels()
		f.Peers[peerId] = peer
	}

	f.mutex.Unlock()
}

// Listen for broadcasts from other devices every 10 seconds
func (f *PeerFinder) listenForBroadcasts(ctx context.Context, d *Downloader) error {
	// Start lisening to the broadcasts of other devices
	entriesChannel := make(chan *mdns.ServiceEntry, 25)
	defer close(entriesChannel)

	go func() {
		for entry := range entriesChannel {
			f.addPeer(ctx, entry, d)
			f.appEvents <- NewMessage[any](PEERS_UPDATED, nil)
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

func (f *PeerFinder) Run(ctx context.Context, d *Downloader) error {
	if err := f.broadcastOurService(); err != nil {
		return err
	}

	if err := f.listenForBroadcasts(ctx, d); err != nil {
		return err
	}

	return f.server.Shutdown()
}
