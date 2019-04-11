package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	//"encoding/hex"
)

var (
	// VERSION My current version
	VERSION = "0.0.0"
)

var (
	siblingsAddr                  = "224.0.0.1:9999"
	beaconFreq      time.Duration = 2 //seconds
	siblingTTL      int64         = 5 //seconds
	beaconInterface               = ""

	maxDatagramSize = 128

	cacheIP     = ""
	cachePort   = "8080"
	cacheGCFreq = 3600

	cacheMode = "standalone"

	cacheGetURL        = "cache/get"
	cacheSetURL        = "cache/set"
	cacheRemoveURL     = "cache/remove/:cache_block"
	cacheRemoveKeyURL  = "cache/remove/key"
	cacheRemoveAppURL  = "cache/remove/application"
	cacheRemoveAllURL  = "cache/remove/all"
	cacheStatsURL      = "cache/stats/:stats_type"
	cacheStatslocalURL = "cache/stats/local"
	cacheStatsAllURL   = "cache/stats/all"
	cacheImportURL     = "cache/import"

	// ME amongst my siblings
	ME          = ""
	siblingsMgr *siblingsManager
)

var (
	mainCache *cacheOperator
)

func udpBeacon() {
	addr, err := net.ResolveUDPAddr("udp", siblingsAddr)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Printf("%+v\n", addr)

	c, err := net.DialUDP("udp", nil, addr)
	for {
		c.Write([]byte(ME))
		time.Sleep(beaconFreq * time.Second)
	}
}

func serveMulticastUDP(a string, iface *net.Interface, callback func(*net.UDPAddr, int, []byte)) {

	log.Println("Listening for siblings beacon")

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

func exportcache(sigCh chan os.Signal) {

	s := <-sigCh

	log.Println("Received signal", s)

	siblingsMgr.distributeContent()

	os.Exit(0)
}

func main() {

	ME, _ = os.Hostname()

	var beaconIface *net.Interface

	readConfig()

	if beaconInterface == "" {
		beaconIface = nil
		beaconInterface = "default"
	} else {
		var err error
		beaconIface, err = net.InterfaceByName(beaconInterface)
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Println("Starting Dcached", VERSION, "on", ME, "[ port", cachePort, "]", cacheMode, "mode")
	if cacheMode == "cluster" {
		log.Println("Multicast group", siblingsAddr)
		log.Println("Beacon interval", int(beaconFreq), "seconds")
		log.Println("Siblings TTL", siblingTTL, "seconds")
		log.Println("Max.datagram size", maxDatagramSize)
		log.Println("Beacon network interface", beaconInterface)
	}
	log.Println("Garbage collector interval", cacheGCFreq, "seconds")

	mainCache = newCache()
	siblingsMgr = newSiblingsManager()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	signal.Notify(sigCh, syscall.SIGTERM)

	go exportcache(sigCh)

	if cacheMode == "cluster" {

		go udpBeacon()
		go serveMulticastUDP(siblingsAddr, beaconIface, siblingsMgr.msgHandler)
		//go serveMulticastUDP(SIBLINGS_ADDR, nil, SIBLINGS_MANAGER.MsgHandler)
	}

	router := httprouter.New()

	router.NotFound = notFoundHandler{}
	router.MethodNotAllowed = methodNotAllowedHandler{}

	router.POST("/"+cacheGetURL, cacheGet)
	router.OPTIONS("/"+cacheGetURL, cacheGetDoc)

	router.POST("/"+cacheSetURL, cacheSet)
	router.OPTIONS("/"+cacheSetURL, cacheSetDoc)

	router.OPTIONS("/"+cacheRemoveURL, cacheRemoveDoc)
	router.POST("/"+cacheRemoveURL, cacheRemove)

	router.OPTIONS("/"+cacheStatsURL, cacheStatsHandlerDoc)
	router.GET("/"+cacheStatsURL, cacheStatsHandler)

	router.OPTIONS("/"+cacheImportURL, cacheImportDoc)
	router.POST("/"+cacheImportURL, cacheImport)

	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", cacheIP, cachePort), router))
}
