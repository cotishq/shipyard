package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/db"
	"github.com/cotishq/shipyard/internal/executor"
	"github.com/cotishq/shipyard/internal/metrics"
	"github.com/cotishq/shipyard/internal/observability"
	"github.com/cotishq/shipyard/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	db.Init()
	storage.Init()
	metrics.Init()

	metricsAddr := strings.TrimSpace(os.Getenv("WORKER_METRICS_ADDR"))
	if metricsAddr == "" {
		metricsAddr = ":8083"
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())

		observability.Info("worker metrics server starting", map[string]any{
			"address": metricsAddr,
		})
		if err := http.ListenAndServe(metricsAddr, mux); err != nil {
			log.Fatal("failed to start worker metrics server:", err)
		}
	}()

	observability.Info("worker started", nil)

	for {
		executor.ProcessNextDeployment()
		time.Sleep(5 * time.Second)
	}
}
