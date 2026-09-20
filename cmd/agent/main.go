package main

import (
	"context"
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
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
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
	ctx := context.Background()
	agentConfig := agent.New(ctx, client, baseURL, fl.Key, logger, fl.RateLimit)

	go func() {
		for {
			time.Sleep(time.Duration(fl.ReportInterval) * time.Second)
			batch := agent.CollectBatch(metricStorage)
			agentConfig.Submit(batch)
			metricStorage.ResetCounter("PollCount")
		}
	}()

	go func() {
		for {
			time.Sleep(time.Duration(fl.PollInterval) * time.Second)
			agent.Collect(metricStorage)
		}
	}()

	go func() {
		for {
			time.Sleep(time.Duration(fl.PollInterval) * time.Second)
			err = agent.CollectSystemMetrics(metricStorage)
			if err != nil {
				logger.Error("CollectSystemMetrics error", zap.Error(err))
			}
		}
	}()
	select {}

}

func setFlags() (flags, error) {
	address := flag.String("a", "localhost:8080", "endpoint address")
	reportInterval := flag.Int("r", 10, "report interval")
	pollInterval := flag.Int("p", 2, "poll interval")
	keySecret := flag.String("k", "", "secret key")
	rateLimit := flag.Int("l", 1, "rate limit")
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
	if fl.Key == "" {
		fl.Key = *keySecret
	}
	if fl.RateLimit == 0 {
		fl.RateLimit = *rateLimit
	}

	return fl, nil
}
