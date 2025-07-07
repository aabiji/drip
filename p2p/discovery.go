package p2p

import (
	"fmt"
	"io"
	"log"
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

	deviceId    string
	devicePort  int
	serviceType string

	peerRemovalTimeout time.Duration
	queryFrequency     time.Duration
	server             *mdns.Server
}

func NewPeerDiscovery() PeerDiscovery {
	return PeerDiscovery{
		deviceId:           getDeviceName(),
		devicePort:         getUnusedPort(),
		Peers:              make(map[string]*Peer),
		serviceType:        "_fileshare._tcp.local.",
		peerRemovalTimeout: time.Minute * 3,
		queryFrequency:     time.Second * 10,
	}
}

func (d *PeerDiscovery) broadcastOurService() error {
	isMobileDevice := runtime.GOOS == "android" || runtime.GOOS == "ios"
	txtVars := []string{
		fmt.Sprintf("is_mobile=%t", isMobileDevice),
	}
	hostname := fmt.Sprintf("%s.local.", d.deviceId)

	service, err := mdns.NewMDNSService(
		d.deviceId, d.serviceType, "local.", hostname,
		d.devicePort, []net.IP{getDeviceIP()}, txtVars)
	if err != nil {
		return err
	}

	d.server, err = mdns.NewServer(&mdns.Config{Zone: service})
	return err
}

func (d *PeerDiscovery) addPeer(entry *mdns.ServiceEntry) {
	peerId := strings.Split(entry.Host, ".")[0]
	if peerId == d.deviceId {
		return // ignore ourselves
	}
	isMobile := strings.Split(entry.InfoFields[0], "=")[1] == "true"

	// This will be used for perfect negotiation. Being polite will
	// mean we forego our own offer when we receive an offer from a peer.
	// Being impolite will mean we ignore the peer's offer and continue with
	// our own. This way, we avoid collisions by knowing that only one peer
	// is able to initiate a connection
	polite := peerId < d.deviceId

	d.mutex.Lock()

	_, exists := d.Peers[peerId]
	if exists {
		d.Peers[peerId].lastHeardFrom = time.Now()
	} else {
		peer := &Peer{
			Ip:            entry.AddrV4,
			Id:            peerId,
			IsMobile:      isMobile,
			polite:        polite,
			udpPort:       entry.Port,
			lastHeardFrom: time.Now(),
		}
		peer.CreateConnection()
		peer.RunClientAndServer(d.devicePort)
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

	original := log.Writer()

	for {
		params := mdns.DefaultParams(d.serviceType)
		params.Entries = entriesChannel
		params.Timeout = d.queryFrequency

		log.SetOutput(io.Discard) // disable the logging mdns does
		if err := mdns.Query(params); err != nil {
			return err
		}
		log.SetOutput(original)

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
