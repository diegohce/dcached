package main

import (
	"fmt"
	"os"
	"syscall"
	"os/signal"
    "github.com/julienschmidt/httprouter"
	"net"
    "net/http"
    "log"
	"time"
	//"encoding/hex"
)

var (
	VERSION = "0.0.0"
)

var (
	SIBLINGS_ADDR = "224.0.0.1:9999"
	BEACON_FREQ time.Duration = 2 //seconds
	SIBLING_TTL int64 = 5 //seconds
	CACHE_IP = ""
	CACHE_PORT = "8080"
	//CACHE_GC_FREQ = 3600
	CACHE_GC_FREQ = 5
	CACHE_GET_URL = "cache/get"
	CACHE_SET_URL = "cache/set"
	CACHE_REMOVE_URL = "cache/remove/:cache_block"
	CACHE_REMOVE_KEY_URL = "cache/remove/key"
	CACHE_REMOVE_APP_URL = "cache/remove/application"
	CACHE_REMOVE_ALL_URL = "cache/remove/all"
	CACHE_STATS_URL = "cache/stats/:stats_type"
	CACHE_STATS_LOCAL_URL = "cache/stats/local"
	CACHE_STATS_ALL_URL = "cache/stats/all"
	CACHE_IMPORT_URL = "cache/import"

	ME = ""
	maxDatagramSize = 128
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

func exportcache(sig_ch chan os.Signal) {

	s := <-sig_ch

	log.Println("Received signal",s)

	SIBLINGS_MANAGER.DistributeContent()

	os.Exit(0)
}

func main() {

	ME, _ = os.Hostname()

	log.Println("Starting Dcached", VERSION,"on", ME)

	SIBLINGS_MANAGER = NewSiblingsManager()

	sig_ch := make(chan os.Signal, 1)
	signal.Notify(sig_ch, os.Interrupt)
	signal.Notify(sig_ch, syscall.SIGTERM)
	go exportcache(sig_ch)

	go udpBeacon()
	go serveMulticastUDP(SIBLINGS_ADDR, nil, SIBLINGS_MANAGER.MsgHandler)

    router := httprouter.New()
    router.POST("/"+CACHE_GET_URL, CacheGet)
    router.POST("/"+CACHE_SET_URL, CacheSet)
    router.POST("/"+CACHE_REMOVE_URL, CacheRemove)
    router.GET("/"+CACHE_STATS_URL, CacheStatsHandler)
    router.POST("/"+CACHE_IMPORT_URL, CacheImport)

    log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", CACHE_IP, CACHE_PORT), router))
}


