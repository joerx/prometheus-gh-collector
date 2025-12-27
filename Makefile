REPO ?= joerx/prometheus-gh-collector
SOURCE_URL ?= https://github.com/$(REPO)

VERSION ?= v0.0.1-$(shell git rev-parse --short HEAD)
CHART_VERSION = $(shell cat charts/collector/Chart.yaml | yq -r '.version')

IMG_REGISTRY ?= ghcr.io
IMG_TAG ?= $(REPO):$(VERSION)

OWNER = $(shell echo "$(REPO)" | awk -F/ '{print $$1}')
NAME = $(shell echo "$(REPO)" | awk -F/ '{print $$2}')

.PHONY: default
default: clean build

.PHONY: build
build: out/bin/$(NAME)

.PHONY: clean
clean:
	rm -rf out/

out/bin/$(NAME):
	go build -o out/bin/$(NAME) .

docker-build:
	docker build -t $(IMG_TAG) \
	--label "org.opencontainers.image.source=$(SOURCE_URL)" \
	--label "org.opencontainers.image.description=Prometheus GitHub Collector" \
	--label "org.opencontainers.image.licenses=Apache-2.0" .

docker-push: docker-build
	docker tag $(IMG_TAG) $(IMG_REGISTRY)/$(IMG_TAG)
	docker push $(IMG_REGISTRY)/$(IMG_TAG)

# TODO: Override source url in Chart.yaml
out/charts/$(NAME)-$(CHART_VERSION).tgz:
	mkdir -p out/charts
	helm package charts/collector --destination out/charts --app-version $(VERSION)

helm-package: out/charts/$(NAME)-$(CHART_VERSION).tgz

helm-push: out/charts/$(NAME)-$(CHART_VERSION).tgz
	helm push out/charts/$(NAME)-$(CHART_VERSION).tgz oci://ghcr.io/$(OWNER)/helm-charts
