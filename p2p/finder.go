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

type PeerInfo struct {
	Ip            net.IP
	Id            string
	LastHeardFrom time.Time
	Port          int
}

type PeerFinder struct {
	devicePort     int
	queryFrequency time.Duration
	server         *mdns.Server
	serviceType    string

	peers map[string]PeerInfo
	mu    sync.Mutex

	nodeEvents chan Message
	ctx        context.Context
}

func NewPeerFinder(
	devicePort int, ctx context.Context, nodeEvents chan Message) PeerFinder {
	return PeerFinder{
		serviceType:    "_fileshare._tcp.local.",
		queryFrequency: time.Second * 10,
		peers:          make(map[string]PeerInfo),
		devicePort:     devicePort,
		ctx:            ctx,
		nodeEvents:     nodeEvents,
	}
}

func (f *PeerFinder) broadcastOurService() error {
	hostname := fmt.Sprintf("%s.local.", deviceName())

	service, err := mdns.NewMDNSService(
		deviceName(), f.serviceType, "local.", hostname,
		f.devicePort, []net.IP{deviceIP()}, []string{})
	if err != nil {
		return err
	}

	f.server, err = mdns.NewServer(&mdns.Config{Zone: service})
	return err
}

func (f *PeerFinder) addPeer(entry *mdns.ServiceEntry) {
	peerId := strings.Split(entry.Host, ".")[0]
	if peerId == deviceName() {
		return
	}

	f.mu.Lock()
	if info, exists := f.peers[peerId]; exists {
		info.LastHeardFrom = time.Now()
		f.peers[peerId] = info
	} else {
		info := PeerInfo{
			Ip:            entry.AddrV4,
			Id:            peerId,
			LastHeardFrom: time.Now(),
			Port:          entry.Port,
		}
		f.peers[peerId] = info
		f.nodeEvents <- NewMessage(ADDED_PEER, info)
	}
	f.mu.Unlock()
}

// Listen for broadcasts from other devices every 10 seconds
func (f *PeerFinder) listenForBroadcasts() error {
	// Start lisening to the broadcasts of other devices
	entriesChannel := make(chan *mdns.ServiceEntry, 25)
	defer close(entriesChannel)

	go func() {
		for entry := range entriesChannel {
			f.addPeer(entry)
		}
	}()

	for {
		params := mdns.DefaultParams(f.serviceType)
		params.Entries = entriesChannel
		params.Timeout = f.queryFrequency

		if err := mdns.Query(params); err != nil {
			return err
		}

		// Remove peers we haven't heard from in a while. We don't close
		// the peer connection here, because it would have already been
		// closed since the webrtc connection would be disrupted.
		f.mu.Lock()
		limit := f.queryFrequency * 3
		for key, peer := range f.peers {
			if time.Since(peer.LastHeardFrom) >= limit {
				delete(f.peers, key)
			}
		}
		f.mu.Unlock()

		// Stop looping when we receive a shutdown signal
		select {
		case <-f.ctx.Done():
			return f.ctx.Err()
		default:
			continue
		}
	}
}

func (f *PeerFinder) Run() error {
	if err := f.broadcastOurService(); err != nil {
		return err
	}

	if err := f.listenForBroadcasts(); err != nil {
		return err
	}

	return f.server.Shutdown()
}
