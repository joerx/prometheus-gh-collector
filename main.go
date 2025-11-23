package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	repoName string
	addr     string

	// This should really be a counter but how do we deduplicate across scrapes?
	numGHADeployments = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_deployments_total",
			Help: "Number of GitHub Actions deployments",
		},
		[]string{"repository", "environment"},
	)
)

func init() {
	prometheus.MustRegister(numGHADeployments)

	flag.StringVar(&repoName, "repo", "owner/name", "Name of the Git repo to scrape deployment metrics for")
	flag.StringVar(&addr, "addr", ":9101", "Address to listen on for HTTP requests")

	flag.Parse()
}

func main() {
	// /metrics is handled by promhttp
	http.Handle("/metrics", promhttp.Handler())

	http.Handle("/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		githubEvent := r.Header.Get("X-GitHub-Event")

		if githubEvent == "ping" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
			return
		}

		if githubEvent == "deployment" {

			payload := struct {
				Deployment struct {
					Environment string `json:"environment"`
					Id          int    `json:"id"`
				} `json:"deployment"`
			}{}

			log.Printf("received deployment event, updating metrics for %s", repoName)

			json.NewDecoder(r.Body).Decode(&payload)
			numGHADeployments.WithLabelValues(repoName, payload.Deployment.Environment).Inc()

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))

			return
		}

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "unsupported event: '%s'", githubEvent)
	}))

	log.Printf("starting deployment metrics demo on %s (metrics: /metrics)", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
