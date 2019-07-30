package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/net/ipv4"

	"github.com/gidoBOSSftw5731/log"
	"github.com/tatsushid/go-fastping"
)

var overhead = 28

var mtus = []int{576 - overhead, 1152 - overhead, 1480 - overhead, 4352 - overhead,
	17900 - overhead, 65535 - overhead}

var maxMTU int

type icmpPkt struct {
	pktType, code, checksum, one, identifier, two, sequenceNum, three byte
	payload                                                           []byte
	four                                                              byte
}

func main() {
	log.SetCallDepth(4)

	ip := "192.110.255.55"
	ipSlice := strings.Split(ip, ".")

	oct0, err := strconv.Atoi(ipSlice[0])
	oct1, err := strconv.Atoi(ipSlice[1])
	oct2, err := strconv.Atoi(ipSlice[2])
	oct3, err := strconv.Atoi(ipSlice[3])

	result := 0

	result, err = testMTU(ip)
	if err != nil || result == 0 {
		log.Panicf("Error with MTU! Maximum allowed %v, error: %v", result, err)
	}
	log.Tracef("%v\n%v", result, err)

	maxMTU = result

	fd, _ := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)

	addr := syscall.SockaddrInet4{
		Port: 0,
		Addr: [4]byte{byte(oct0), byte(oct1), byte(oct2), byte(oct3)},
	}

	p := pkt()

	err = syscall.Sendto(fd, p, 0, &addr)
	if err != nil {
		log.Panicln(err)
	}

}

//testMTU is a function to test which MTUs are available to use.
//err will still return nil but result will be 0 if none is available
func testMTU(ip string) (int, error) {
	result := 0
	var err error

	for _, mtu := range mtus {
		p := fastping.NewPinger()

		ra, err := net.ResolveIPAddr("ip4:icmp", ip)
		if err != nil {
			log.Errorln("error resolving ip addr, maybe try failover IP? ", err)
			return result, err
		}

		p.AddIPAddr(ra)

		p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
			//log.Tracef("MTU %v SUCCESS", mtu)
			result = mtu
		}

		p.Size = mtu

		err = p.Run()

		if err != nil {
			log.Debugln(err)
			continue
		}
	}

	return result, err
}

//pkt is a function to make an ICMP packet.
func pkt() []byte {
	h := ipv4.Header{
		Version:  4,
		Len:      20,
		TotalLen: 20 + 10, // 20 bytes for IP, 10 for ICMP
		TTL:      64,
		Protocol: 1, // ICMP
		Dst:      net.IPv4(127, 0, 0, 1),
		// ID, Src and Checksum will be set for us by the kernel
	}

	payload := []byte("foofoofoo")

	var icmp = icmpPkt{
		8,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		payload,
		0xDE}
	//cs := csum([]byte(fmt.Sprint(icmp)))
	cs := csum(icmp)
	icmp.checksum = byte(cs)
	icmp.one = byte(cs >> 8)

	out, err := h.Marshal()
	if err != nil {
		log.Fatal(err)
	}
	return append(out, []byte(fmt.Sprint(icmp))...)
}


