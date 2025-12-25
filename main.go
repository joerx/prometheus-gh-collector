package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	addr          string
	webhookSecret string

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
	flag.StringVar(&webhookSecret, "webhook-secret", "", "GitHub webhook secret")

	flag.Parse()
}

// eventHandler handles incoming GitHub webhook events
func eventHandler(w http.ResponseWriter, r *http.Request) {
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

	w.Write([]byte("ok"))
}

func logger(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		h.ServeHTTP(w, r)
	}
}

func ghValidate(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		valid, err := validateGhPayload(r)

		if err != nil {
			log.Printf("msg='could not validate GitHub payload' err=%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("could not validate payload"))
		}

		if !valid {
			log.Printf("msg='invalid GitHub webhook signature'")

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid signature"))
		}

		h.ServeHTTP(w, r)
	}
}

func validateGhPayload(r *http.Request) (bool, error) {
	if webhookSecret == "" {
		return false, fmt.Errorf("No WEBHOOK_SECRET configured")
	}

	// Read the body so we can compute the HMAC, then restore it for downstream handlers
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, fmt.Errorf("could not read request body: %v", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	expected := "sha256=" + computeHMAC256(body, webhookSecret)
	received := r.Header.Get("X-Hub-Signature-256")
	return hmac.Equal([]byte(received), []byte(expected)), nil
}

// computeHMAC256 computes the Keyed-Hash Message Authentication Code (HMAC) of data using the given secret using a SHA-256 hash function
func computeHMAC256(data []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func main() {
	if webhookSecret == "" {
		log.Fatalf("WEBHOOK_SECRET must be set")
	}

	// /metrics is handled by promhttp
	http.Handle("/metrics", logger(promhttp.Handler()))

	// /events handles GitHub webhooks
	http.Handle("/events", logger(ghValidate(http.HandlerFunc(eventHandler))))

	log.Printf("starting deployment metrics demo on %s (metrics: /metrics)", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
