package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type statsAll struct {
	TotalBytes int64
	Nodes      []*cacheStats
}

func (sta *statsAll) String() string {
	b, _ := json.Marshal(sta)
	return string(b)
}
func (sta *statsAll) write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", sta)
}

func (c *cacheStats) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}
func (c *cacheStats) write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", c)
}

func cacheStatsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	statsop := &statsOp{
		done: make(chan bool),
	}

	statsType := ps.ByName("stats_type")

	log.Println("request::stats type", statsType)

	timeitStart := time.Now().UnixNano()

	mainCache.CacheStats <- statsop
	<-statsop.done

	if statsType == "local" {
		statsop.stats.ElapsedNs = time.Now().UnixNano() - timeitStart
		statsop.stats.write(w)

	} else if statsType == "all" {
		sta := &statsAll{}
		sta.Nodes = append(sta.Nodes, statsop.stats)
		sta.TotalBytes = statsop.stats.MemBytes

		ch := siblingsMgr.propagateStats()
		for stat := range ch {
			if stat != nil {
				sta.Nodes = append(sta.Nodes, stat)
				sta.TotalBytes += stat.MemBytes
			}
		}
		statsop.stats.ElapsedNs = time.Now().UnixNano() - timeitStart
		sta.write(w)

	} else {
		e := newException("ResourceNotFoundException", "Resource not found")
		e.write(w)
	}

}
