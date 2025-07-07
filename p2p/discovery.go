package p2p

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

type PeerDiscovery struct {
	Peers map[string]*Peer
	mutex sync.Mutex // guards Peers

	serviceType        string
	peerRemovalTimeout time.Duration
	queryFrequency     time.Duration
	server             *mdns.Server
}

func NewPeerDiscovery() PeerDiscovery {
	return PeerDiscovery{
		serviceType: "_fileshare._tcp.local.",
		Peers:       make(map[string]*Peer),

		peerRemovalTimeout: time.Minute * 3,
		queryFrequency:     time.Second * 10,
	}
}

func (d *PeerDiscovery) broadcastOurService() error {
	isMobileDevice := runtime.GOOS == "android" || runtime.GOOS == "ios"
	txtVars := []string{
		fmt.Sprintf("is_mobile=%t", isMobileDevice),
	}
	hostname := fmt.Sprintf("%s.local.", DEVICE_ID)

	service, err := mdns.NewMDNSService(
		DEVICE_ID, d.serviceType, "local.", hostname,
		DEVICE_PORT, []net.IP{getDeviceIP()}, txtVars)
	if err != nil {
		return err
	}

	d.server, err = mdns.NewServer(&mdns.Config{Zone: service})
	return err
}

func (d *PeerDiscovery) addPeer(entry *mdns.ServiceEntry) {
	peerId := strings.Split(entry.Host, ".")[0]
	if peerId == DEVICE_ID {
		return // ignore ourselves
	}
	isMobile := strings.Split(entry.InfoFields[0], "=")[1] == "true"

	d.mutex.Lock()

	_, exists := d.Peers[peerId]
	if exists {
		d.Peers[peerId].lastHeardFrom = time.Now()
	} else {
		peer := &Peer{
			Ip:            entry.AddrV4,
			Id:            peerId,
			IsMobile:      isMobile,
			udpPort:       entry.Port,
			lastHeardFrom: time.Now(),
		}
		peer.CreateConnection()
		peer.RunClientAndServer()
		peer.SetupDataChannel()
		d.Peers[peerId] = peer
	}

	d.mutex.Unlock()
}

// Listen for broadcasts from other devices every 10 seconds
func (d *PeerDiscovery) listenForBroadcasts(eventChannel chan int) error {
	// Start lisening to the broadcasts of other devices
	entriesChannel := make(chan *mdns.ServiceEntry, 25)
	go func() {
		for entry := range entriesChannel {
			d.addPeer(entry)
		}
	}()

	for {
		params := mdns.DefaultParams(d.serviceType)
		params.Entries = entriesChannel
		params.Timeout = d.queryFrequency
		if err := mdns.Query(params); err != nil {
			return err
		}

		// Remove peers we haven't heard from in a while
		d.mutex.Lock()
		for key, value := range d.Peers {
			if time.Since(value.lastHeardFrom) >= d.peerRemovalTimeout {
				delete(d.Peers, key)
			}
		}
		d.mutex.Unlock()

		select {
		case event := <-eventChannel:
			if event == QUIT {
				close(entriesChannel)
				return nil
			}
		default:
			continue
		}
	}
}

func (d *PeerDiscovery) Run(eventChannel chan int) error {
	if err := d.broadcastOurService(); err != nil {
		return err
	}

	if err := d.listenForBroadcasts(eventChannel); err != nil {
		return err
	}

	return d.server.Shutdown()
}
