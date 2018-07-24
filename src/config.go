package main


import (
	"github.com/creamdog/gonfig"
	"os"
	"log"
	"time"
)





func readConfig() {

	f, err := os.Open("dcached.conf")
	if err != nil {
		log.Println("config::readConfig", err)
		log.Println("config::readConfig Will use default values")
		return
	}
	defer f.Close()

	config, err := gonfig.FromYml(f)
	if err != nil {
		log.Println("config::readConfig", err)
		log.Println("config::readConfig Will use default values")
		return
	}

	var (
		sibling_ttl int
		beacon_freq int
	)


	SIBLINGS_ADDR, _ = config.GetString("siblings/address", "224.0.0.1:9999")

	beacon_freq, _   = config.GetInt("siblings/beacon_freq", 2)
	BEACON_FREQ = time.Duration(beacon_freq)

	sibling_ttl, _   = config.GetInt("siblings/ttl", 5)
	SIBLING_TTL = int64(sibling_ttl)

	BEACON_INTERFACE, _ = config.GetString("siblings/beacon_interface", "")

	maxDatagramSize, _ = config.GetInt("siblings/max_datagram_size", 128)

	CACHE_IP, _      = config.GetString("cache/ip", "")
	CACHE_PORT, _    = config.GetString("cache/port", "8080")
	CACHE_GC_FREQ, _ = config.GetInt("cache/gc_freq", 3600)

}



