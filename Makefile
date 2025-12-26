REPO = joerx/prometheus-gh-collector
SOURCE_URL ?= https://github.com/$(REPO)
VERSION ?= v0.0.1-$(shell git rev-parse --short HEAD)

IMG_REGISTRY ?= ghcr.io
IMG_TAG ?= $(REPO):$(VERSION)

.PHONY: default
default: clean build

.PHONY: build
build: bin/$(NAME)

.PHONY: clean
clean:
	rm -rf bin

bin/$(NAME):
	go build -o bin/$(NAME) .

docker-build: bin/$(NAME)
	docker build -t $(IMG_TAG) \
	--label "org.opencontainers.image.source=$(SOURCE_URL)" \
	--label "org.opencontainers.image.description=Prometheus GitHub Collector" \
	--label "org.opencontainers.image.licenses=Apache-2.0" .

docker-push: docker-build
	docker tag $(IMG_TAG) $(IMG_REGISTRY)/$(IMG_TAG)
	docker push $(IMG_REGISTRY)/$(IMG_TAG)
