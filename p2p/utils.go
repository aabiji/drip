package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"os"
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

func getDeviceName(debugMode bool) string {
	if debugMode {
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
	// get the os to give a random free port
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}
