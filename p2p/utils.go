package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
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

func getUnusedPort() int {
	// get the os to give us a random unused address
	addr := &net.UDPAddr{Port: 0, IP: net.IPv4zero}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

var DEVICE_ID = getDeviceName()
var DEVICE_PORT = getUnusedPort()

const QUIT = 0 // Event channel value
