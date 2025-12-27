REPO ?= joerx/prometheus-gh-collector
SOURCE_URL ?= https://github.com/$(REPO)

VERSION ?= v0.0.1-$(shell git rev-parse --short HEAD)
CHART_VERSION ?= $(VERSION)
# CHART_VERSION = $(shell cat charts/collector/Chart.yaml | yq -r '.version')

IMG_REGISTRY ?= ghcr.io
IMG_TAG ?= $(REPO):$(VERSION)

OWNER = $(shell echo "$(REPO)" | awk -F/ '{print $$1}')
NAME = $(shell echo "$(REPO)" | awk -F/ '{print $$2}')

RELEASE_BRANCH ?= main

.PHONY: default
default: clean build

.PHONY: build
build: out/bin/$(NAME)

.PHONY: clean
clean:
	rm -rf out/

.PHONY: release
release:
	gh release create $(VERSION) --title "Release $(VERSION)" --target $(RELEASE_BRANCH) --generate-notes

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

.PHONY: helm-package
helm-package: out/charts/$(NAME)-$(CHART_VERSION).tgz:out/charts/$(NAME)-$(CHART_VERSION).tgz

out/charts/$(NAME)-$(CHART_VERSION).tgz:
	mkdir -p out/charts
	cp -r charts/collector out/charts/$(NAME)-$(CHART_VERSION)
	yq -i '.annotations["org.opencontainers.image.source"] = "$(SOURCE_URL)"' out/charts/$(NAME)-$(CHART_VERSION)/Chart.yaml
	yq -i '.version = "$(VERSION)"' out/charts/$(NAME)-$(CHART_VERSION)/Chart.yaml
	helm package out/charts/$(NAME)-$(CHART_VERSION) --destination out/charts --app-version $(VERSION)

out/charts/$(NAME)-$(CHART_VERSION).tgz: out/charts/$(NAME)-$(CHART_VERSION)

.PHONY: helm-push
helm-push: helm-package
	helm push out/charts/$(NAME)-$(CHART_VERSION).tgz oci://ghcr.io/$(OWNER)/helm-charts
