package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/BREJJNEVV/metrics/internal/agent"
)

type flags struct {
	address        string
	reportInterval int
	pollInterval   int
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fl := setFlags()

	client := &http.Client{}
	metricStorage := agent.CreateMetricStorage()
	baseURL := fmt.Sprintf("http://%s", fl.address)

	go func() {
		for {
			time.Sleep(time.Duration(fl.reportInterval) * time.Second)
			agent.Send(metricStorage, client, baseURL)
			metricStorage.ResetCounter("PollCount")
		}
	}()

	go func() {
		for {
			time.Sleep(time.Duration(fl.pollInterval) * time.Second)
			agent.Collect(metricStorage)
		}
	}()

	select {}

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
