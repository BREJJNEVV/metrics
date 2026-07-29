package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/BREJJNEVV/metrics/internal/agent"
)

type flags struct {
	address        string
	reportInterval int
	pollInterval   int
}

func main() {

	fl := setFlags()

	client := &http.Client{}
	metricStorage := agent.CreateMetricStorage()
	baseURL := fmt.Sprintf("http://%s", fl.address)

	reportStart := false
	var mu sync.Mutex
	go func() {
		for {
			time.Sleep(time.Duration(fl.reportInterval) * time.Second)
			mu.Lock()
			reportStart = true
			mu.Unlock()
		}
	}()

	for {
		agent.Collect(metricStorage)
		time.Sleep(time.Duration(fl.pollInterval) * time.Second)
		mu.Lock()
		rst := reportStart
		reportStart = false
		mu.Unlock()
		if rst == true {
			agent.Send(metricStorage, client, baseURL)
		}
	}

}

func setFlags() flags {
	address := flag.String("a", "localhost:8080", "endpoint address")
	reportInterval := flag.Int("r", 10, "report interval")
	pollInterval := flag.Int("p", 2, "poll interval")
	flag.Parse()
	fl := flags{
		address:        *address,
		reportInterval: *reportInterval,
		pollInterval:   *pollInterval,
	}
	return fl
}
