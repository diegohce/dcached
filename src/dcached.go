package main

import (
	"os"
    "github.com/julienschmidt/httprouter"
	"net"
    "net/http"
    "log"
	"time"
	//"encoding/hex"
)


var (
	SIBLINGS_ADDR = "224.0.0.1:9999"
	BEACON_FREQ time.Duration = 2 //seconds
	SIBLING_TTL int64 = 10 //seconds
	CACHE_IP_PORT = ":8080"
	CACHE_GC_FREQ = 3600
	MY_PORT = "8080"

	ME = ""
	maxDatagramSize = 8192
	SIBLINGS_MANAGER *SiblingsManager
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

	SIBLINGS_MANAGER = NewSiblingsManager()

	go udpBeacon()
	go serveMulticastUDP(SIBLINGS_ADDR, nil, SIBLINGS_MANAGER.MsgHandler)

    router := httprouter.New()
    router.POST("/cache/get", CacheGet)
    router.POST("/cache/set", CacheSet)

    log.Fatal(http.ListenAndServe(CACHE_IP_PORT, router))
}


