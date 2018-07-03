package main

import (
	"os"
    "fmt"
    "github.com/julienschmidt/httprouter"
	"net"
    "net/http"
    "log"
	"time"
	//"encoding/hex"
)



type Sibling struct {
	Node     string
	Ip       string
	LastCall int64
}


var (
	SIBLINGS_ADDR = "224.0.0.1:9999"
	BEACON_FREQ time.Duration = 2 //seconds
	SIBLING_TTL int64 = 10 //seconds
	CACHE_IP_PORT = ":8080"
	CACHE_GC_FREQ = 3600

	ME = ""
	maxDatagramSize = 8192
	Siblings = map[string]Sibling{}
)



func udpBeacon() {
	addr, err := net.ResolveUDPAddr("udp", SIBLINGS_ADDR)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%+v\n", addr)

	c, err := net.DialUDP("udp", nil, addr)
	for {
		c.Write([]byte(ME))
		time.Sleep(BEACON_FREQ * time.Second)
	}
}

func msgHandler(src *net.UDPAddr, bcount int, bread []byte) {
	log.Println(bcount, "bytes read from", src)
	//log.Println(hex.Dump(b[:n]))

	node := string(bread[:bcount])

	if node != ME {
		Siblings[node] = Sibling{Node: node,
								 Ip: fmt.Sprintf("%s", src.IP),
								 LastCall: time.Now().Unix() }
	}

	fmt.Println("Registered siblings")
	fmt.Printf("%+v\n", Siblings)

	for k, sibling := range Siblings {
		if time.Now().Unix() - sibling.LastCall >= SIBLING_TTL {
			fmt.Println(k ,"must die")
			delete(Siblings, k)
		}

	}
}


func serveMulticastUDP(a string, iface *net.Interface, callback func(*net.UDPAddr, int, []byte)) {
	addr, err := net.ResolveUDPAddr("udp", a)
	if err != nil {
		log.Fatal(err)
	}
	l, err := net.ListenMulticastUDP("udp", iface, addr)
	if err != nil {
		log.Fatal(err)
	}
	l.SetReadBuffer(maxDatagramSize)
	for {
		bread := make([]byte, maxDatagramSize)
		bcount, src, err := l.ReadFromUDP(bread)
		if err != nil {
			log.Fatal("ReadFromUDP failed:", err)
		}
		callback(src, bcount, bread)
	}
}


func main() {

	ME, _ = os.Hostname()

	go udpBeacon()
	go serveMulticastUDP(SIBLINGS_ADDR, nil, msgHandler)

    router := httprouter.New()
    router.GET("/", Index)
    router.GET("/hello/:name", Hello)

    router.POST("/cache/get", CacheGet)
    router.POST("/cache/set", CacheSet)

    log.Fatal(http.ListenAndServe(CACHE_IP_PORT, router))
}


