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
	ip            net.IP
	id            string
	lastHeardFrom time.Time
	port          int
}

type PeerFinder struct {
	devicePort         int
	peerRemovalTimeout time.Duration
	queryFrequency     time.Duration
	server             *mdns.Server
	serviceType        string

	peers map[string]PeerInfo
	mu    sync.Mutex

	ctx               context.Context
	addPeerHandler    func(PeerInfo)
	removePeerHandler func(string)
}

func NewPeerFinder(
	devicePort int, ctx context.Context,
	addPeerHandler func(PeerInfo),
	removePeerHandler func(string),
) PeerFinder {
	return PeerFinder{
		serviceType:        "_fileshare._tcp.local.",
		peerRemovalTimeout: time.Second * 15,
		queryFrequency:     time.Second * 10,
		peers:              make(map[string]PeerInfo),
		devicePort:         devicePort,
		ctx:                ctx,
		addPeerHandler:     addPeerHandler,
		removePeerHandler:  removePeerHandler,
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
		info.lastHeardFrom = time.Now()
		f.peers[peerId] = info
	} else {
		info := PeerInfo{
			ip:            entry.AddrV4,
			id:            peerId,
			lastHeardFrom: time.Now(),
			port:          entry.Port,
		}
		f.peers[peerId] = info
		f.addPeerHandler(info)
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

		// Remove peers we haven't heard from in a while
		f.mu.Lock()
		for key, peer := range f.peers {
			if time.Since(peer.lastHeardFrom) >= f.peerRemovalTimeout {
				f.removePeerHandler(key)
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
