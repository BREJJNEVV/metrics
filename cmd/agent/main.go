package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/BREJJNEVV/metrics/internal/agent"
)

const collectingTime int = 2
const sendingTime int = 10

func main() {

	client := &http.Client{}
	port := "8080"
	metricStorage := agent.CreateMetricStorage()
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	//TODO: добавить строгий таймер, обернуть мютексами
	for {
		for range sendingTime / collectingTime {
			agent.Collect(metricStorage)
			time.Sleep(time.Duration(collectingTime) * time.Second)
		}
		agent.Send(metricStorage, client, baseURL)
	}

}
