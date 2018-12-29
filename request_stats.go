package main

import (
    "fmt"
	"log"
	"time"
    "net/http"
	"encoding/json"
    "github.com/julienschmidt/httprouter"
)

type StatsAll struct {
	TotalBytes int64
	Nodes []*CacheStats
}
func (sta *StatsAll) String() string {
	b, _ := json.Marshal(sta)
	return string(b)
}
func (sta *StatsAll) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", sta)
}


func (c *CacheStats) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}
func (c *CacheStats) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", c)
}


func CacheStatsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	statsop := &statsOp{
		done: make(chan bool),
	}

	stats_type := ps.ByName("stats_type")

	log.Println("request::stats type", stats_type)

	timeit_start := time.Now().UnixNano()

	CACHE.CacheStats <-statsop
	<-statsop.done

	if stats_type == "local" {
		statsop.stats.Elapsed_ns = time.Now().UnixNano() - timeit_start
		statsop.stats.Write(w)

	} else if stats_type == "all" {
		sta := &StatsAll{}
		sta.Nodes = append(sta.Nodes, statsop.stats)
		sta.TotalBytes = statsop.stats.MemBytes

		ch := SIBLINGS_MANAGER.PropagateStats()
		for stat := range ch {
			if stat != nil {
				sta.Nodes = append(sta.Nodes, stat)
				sta.TotalBytes += stat.MemBytes
			}
		}
		statsop.stats.Elapsed_ns = time.Now().UnixNano() - timeit_start
		sta.Write(w)


	} else {
		e := NewException("ResourceNotFoundException", "Resource not found")
		e.Write(w)
	}

}

