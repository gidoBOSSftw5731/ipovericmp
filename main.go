package main

import (
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"./iana" // golang's STUPID internal tag forces me to do this instead of accessing https://godoc.org/golang.org/x/net/internal/iana directly
	"github.com/gidoBOSSftw5731/log"
	"github.com/tatsushid/go-fastping"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

var overhead = 28

var mtus = []int{576 - overhead, 1152 - overhead, 1480 - overhead, 4352 - overhead,
	17900 - overhead, 65535 - overhead}

var maxMTU int

func main() {
	log.SetCallDepth(4)

	ip := "192.110.255.55"
	/*ipSlice := strings.Split(ip, ".")

	oct0, err := strconv.Atoi(ipSlice[0])
	oct1, err := strconv.Atoi(ipSlice[1])
	oct2, err := strconv.Atoi(ipSlice[2])
	oct3, err := strconv.Atoi(ipSlice[3]) */

	result := 0

	result, err := testMTU(ip)
	if err != nil || result == 0 {
		log.Panicf("Error with MTU! Maximum allowed %v, error: %v", result, err)
	}
	log.Tracef("%v\n%v", result, err)

	maxMTU = result

	var wg sync.WaitGroup
	wg.Add(1)

	go icmpListen(&wg)

	b := randomPayload()
	sendEcho(ip, string(b))

	wg.Wait()

}

func icmpListen(wg *sync.WaitGroup) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	for {
		var msg []byte
		length, sourceIP, err := conn.ReadFrom(msg)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf("message = '%s', length = %d, source-ip = %s", string(msg), length, sourceIP)
		//wg.Done()
	}
}

func randomPayload() []rune {
	b := make([]rune, 1024)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}

func sendEcho(ip, payload string) {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatalf("listen err, %s", err)
	}
	defer c.Close()

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte(payload),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := c.WriteTo(wb, &net.IPAddr{IP: net.ParseIP(ip)}); err != nil {
		log.Fatalf("WriteTo err, %s", err)
	}

	rb := make([]byte, 1500)
	n, peer, err := c.ReadFrom(rb)
	if err != nil {
		log.Fatal(err)
	}
	rm, err := icmp.ParseMessage(iana.ProtocolICMP, rb[:n])
	if err != nil {
		log.Fatal(err)
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		log.Printf("got reflection from %v", peer)
	default:
		log.Printf("got %+v; want echo reply", rm)
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
