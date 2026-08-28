package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/BREJJNEVV/metrics/internal/agent"
	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
)

type flags struct {
	Address        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	fl, err := setFlags()
	if err != nil {
		logger.Fatal("fatal error", zap.Error(err))
	}
	client := &http.Client{}
	metricStorage := agent.CreateMetricStorage()
	baseURL := fmt.Sprintf("http://%s", fl.Address)

	go func() {
		for {
			time.Sleep(time.Duration(fl.ReportInterval) * time.Second)
			agent.Send(metricStorage, client, baseURL, logger)
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

func setFlags() (flags, error) {
	address := flag.String("a", "localhost:8080", "endpoint address")
	reportInterval := flag.Int("r", 10, "report interval")
	pollInterval := flag.Int("p", 2, "poll interval")
	flag.Parse()

	fl := flags{}
	err := env.Parse(&fl)
	if err != nil {
		return fl, err
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

	return fl, nil
}
