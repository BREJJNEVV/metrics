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
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fl := setFlags()
	client := &http.Client{}
	metricStorage := agent.CreateMetricStorage()
	baseURL := fmt.Sprintf("http://%s", fl.Address)

	go func() {
		for {
			time.Sleep(time.Duration(fl.ReportInterval) * time.Second)
			agent.Send(metricStorage, client, baseURL)
			metricStorage.ResetCounter("PollCount")
		}
	}()

	go func() {
		for {
			time.Sleep(time.Duration(fl.PollInterval) * time.Second)
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

	fl := flags{}
	err := env.Parse(&fl)
	if err != nil {
		log.Fatal(err)
	}

	if fl.Address == "" {
		fl.Address = *address
	}
	if fl.ReportInterval == 0 {
		fl.ReportInterval = *reportInterval
	}
	if fl.PollInterval == 0 {
		fl.PollInterval = *pollInterval
	}

	return fl
}
