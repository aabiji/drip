package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/mdns"
)

func getDeviceIP() net.IP {
	// get the prefered outbound ip address of this device
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		panic(err) // getting the ip is a must
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}

// if the DRIP_DEBUG environment variable is set, this will return
// a ranomized name instead of the actual host name to facilitate testing
func getDeviceName() string {
	debug := strings.Trim(os.Getenv("DRIP_DEBUG"), " ")
	if len(debug) > 0 {
		id := rand.Intn(1000000)
		return fmt.Sprintf("peer-%d", id)
	}

	name, err := os.Hostname()
	if err != nil {
		panic(err) // getting the hostname is a must
	}
	return name
}

const QUIT = 0 // Event channel value

var DEVICE_ID = getDeviceName()

type PeerDiscovery struct {
	Peers map[string]Peer
	mutex sync.Mutex // guards Peers

	serviceType        string
	peerRemovalTimeout time.Duration
	queryFrequency     time.Duration
	server             *mdns.Server
}

func NewPeerDiscovery() PeerDiscovery {
	return PeerDiscovery{
		serviceType: "_fileshare._tcp.local.",
		Peers:       make(map[string]Peer),

		peerRemovalTimeout: time.Minute * 3,
		queryFrequency:     time.Second * 10,
	}
}

func (d *PeerDiscovery) broadcastOurService() error {
	isMobileDevice := runtime.GOOS == "android" || runtime.GOOS == "ios"
	txtVar := fmt.Sprintf("is_mobile=%t", isMobileDevice)
	hostname := fmt.Sprintf("%s.local.", DEVICE_ID)

	service, err := mdns.NewMDNSService(
		DEVICE_ID, d.serviceType, "local.",
		hostname, 8080, []net.IP{getDeviceIP()},
		[]string{txtVar})
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

	peer := Peer{
		Ip:            entry.AddrV4,
		Id:            peerId,
		IsMobile:      isMobile,
		lastHeardFrom: time.Now(),
	}

	// TODO: test this!
	d.mutex.Lock()
	existing, exists := d.Peers[peerId]
	if exists {
		existing.lastHeardFrom = time.Now()
		d.Peers[peerId] = existing
	} else {
		peer.CreateConnection()
		peer.RunClientAndServer()
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
