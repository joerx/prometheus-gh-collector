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

type deploymentEvent struct {
	Deployment deployment `json:"deployment"`
	Repository repository `json:"repository"`
}

type deployment struct {
	Environment string `json:"environment"`
	Id          int    `json:"id"`
}

type deploymentStatusEvent struct {
	DeploymentStatus deploymentStatus `json:"deployment_status"`
	Deployment       deployment       `json:"deployment"`
	Repository       repository       `json:"repository"`
}

type deploymentStatus struct {
	State string `json:"state"`
}

type repository struct {
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	HtmlURL  string `json:"html_url"`
}

var (
	addr string

	// This should really be a counter but how do we deduplicate across scrapes?
	ctrGHDeployments = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gh_deployments_total",
			Help: "Number of GitHub Actions deployments by environment",
		},
		[]string{"repository", "environment"},
	)

	ctrGHDeploymentStatuses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gh_deployment_status_total",
			Help: "Number of GitHub Actions status event by state and environment",
		},
		[]string{"repository", "environment", "state"},
	)
)

func init() {
	prometheus.MustRegister(ctrGHDeployments)
	prometheus.MustRegister(ctrGHDeploymentStatuses)

	flag.StringVar(&addr, "addr", ":9101", "Address to listen on for HTTP requests")

	flag.Parse()
}

func main() {
	// /metrics is handled by promhttp
	http.Handle("/metrics", promhttp.Handler())

	// /events handles GitHub webhooks
	http.Handle("/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		githubEvent := r.Header.Get("X-GitHub-Event")

		log.Printf("msg='received GitHub event' event=%s", githubEvent)

		if githubEvent == "ping" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
			return
		}

		if githubEvent == "deployment" {
			e := deploymentEvent{}

			if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "could not decode payload: %v", err)
				return
			}

			log.Printf("msg='received deployment event' environment=%s repo=%s", e.Deployment.Environment, e.Repository.FullName)
			ctrGHDeployments.WithLabelValues(e.Repository.FullName, e.Deployment.Environment).Inc()
		}

		if githubEvent == "deployment_status" {
			e := deploymentStatusEvent{}

			if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "could not decode payload: %v", err)
				return
			}

			log.Printf("msg='received deployment status event' state=%s environment=%s repo=%s", e.DeploymentStatus.State, e.Deployment.Environment, e.Repository.FullName)
			ctrGHDeploymentStatuses.WithLabelValues(e.Repository.FullName, e.Deployment.Environment, e.DeploymentStatus.State).Inc()
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	log.Printf("starting deployment metrics demo on %s (metrics: /metrics)", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
