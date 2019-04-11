package main

import (
	"log"
	"os"
	"time"

	"github.com/creamdog/gonfig"
)

func readConfig() {

	f, err := os.Open("/etc/dcached.conf")
	if err != nil {
		log.Println("config::readConfig", err)
		log.Println("config::readConfig Will try ./dcached.conf")
	}

	if f == nil {
		f, err = os.Open("dcached.conf")
		if err != nil {
			log.Println("config::readConfig", err)
			log.Println("config::readConfig Will use default values")
			return
		}
	}
	defer f.Close()

	config, err := gonfig.FromYml(f)
	if err != nil {
		log.Println("config::readConfig", err)
		log.Println("config::readConfig Will use default values")
		return
	}

	var (
		sTTL  int
		bFreq int
	)

	siblingsAddr, _ = config.GetString("siblings/address", "224.0.0.1:9999")

	bFreq, _ = config.GetInt("siblings/beacon_freq", 2)
	beaconFreq = time.Duration(bFreq)

	sTTL, _ = config.GetInt("siblings/ttl", 5)
	siblingTTL = int64(sTTL)

	beaconInterface, _ = config.GetString("siblings/beacon_interface", "")

	maxDatagramSize, _ = config.GetInt("siblings/max_datagram_size", 128)

	cacheIP, _ = config.GetString("cache/ip", "")
	cachePort, _ = config.GetString("cache/port", "8080")
	cacheGCFreq, _ = config.GetInt("cache/gc_freq", 3600)

	cacheMode, _ = config.GetString("cache/mode", "standalone")

}
