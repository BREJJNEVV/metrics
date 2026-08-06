package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/BREJJNEVV/metrics/internal/agent"
	"github.com/caarlos0/env/v6"
)

type flags struct {
	address        string `env:"ADDRESS"`
	reportInterval int    `env:"REPORT_INTERVAL"`
	pollInterval   int    `env:"POLL_INTERVAL"`
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
	fl := flags{}
	err := env.Parse(&fl)
	if err != nil {
		log.Fatal(err)
	}
	if fl.address == "" {
		fl.address = *flag.String("a", "localhost:8080", "endpoint address")
	}
	if fl.reportInterval == 0 {
		fl.reportInterval = *flag.Int("r", 10, "report interval")
	}
	if fl.pollInterval == 0 {
		fl.pollInterval = *flag.Int("p", 2, "poll interval")
	}

	flag.Parse()

	return fl
}
