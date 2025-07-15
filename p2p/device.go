package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"os"
)

const DEBUG_MODE bool = true

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

var cached_device_name string = ""

func getDeviceName() string {
	if len(cached_device_name) > 0 {
		return cached_device_name
	}

	if DEBUG_MODE {
		id := rand.Intn(1000000)
		cached_device_name = fmt.Sprintf("peer-%d", id)
		return cached_device_name
	}

	name, err := os.Hostname()
	if err != nil {
		panic(err) // getting the hostname is a must
	}
	cached_device_name = name
	return name
}

func getUnusedPort() int {
	// get the os to give a random free port
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}
