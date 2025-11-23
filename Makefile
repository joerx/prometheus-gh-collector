# VERSION ?= $(shell git rev-parse --short HEAD)
# IMG ?= joerx/prometheus-gha-collector:$(VERSION)
DOCKER_HOST ?= ghcr.io

DOCKER_REPO := $(DOCKER_HOST)/$(IMG)

.PHONY: default
default: clean build

.PHONY: build
build: out/prometheus-gha-collector

.PHONY: clean
clean:
	rm -rf out

out/prometheus-gha-collector:
	go build -o out/prometheus-gha-collector .
