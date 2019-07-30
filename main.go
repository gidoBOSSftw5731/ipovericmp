package main

import (
	"net"
	"time"

	"github.com/gidoBOSSftw5731/log"
	"github.com/tatsushid/go-fastping"
)

var overhead = 28

var mtus = [6]int{576 - overhead, 1152 - overhead, 1480 - overhead, 4352 - overhead,
	17900 - overhead, 65535 - overhead}

var maxMTU int

func main() {
	log.SetCallDepth(4)

	result := 0

	result, err := testMTU("imagen.click")
	if err != nil || result == 0 {
		log.Panicf("Error with MTU! Maximum allowed %v, error: %v", result, err)
	}
	log.Tracef("%v\n%v", result, err)

	maxMTU = result

}

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
