NAME = prometheus-gh-collector
OWNER = joerx
REPO ?= https://github.com/$(OWNER)/$(NAME)
VERSION ?= $(shell git rev-parse --short HEAD)

IMG_BASE ?= $(NAME)
IMG_TAG ?= $(IMG_BASE):$(VERSION)
IMG_REPO ?= ghcr.io/$(OWNER)/$(IMG_BASE)

.PHONY: default
default: clean build

.PHONY: build
build: out/$(NAME)

.PHONY: clean
clean:
	rm -rf out

out/$(NAME):
	go build -o out/$(NAME) .

docker-build: out/$(NAME)
	docker build -t $(IMG_TAG) \
	--label "org.opencontainers.image.source=$(REPO)" \
	--label "org.opencontainers.image.description=Prometheus GitHub Collector" \
	--label "org.opencontainers.image.licenses=Apache-2.0" .

docker-push: docker-build
	docker tag $(IMG_TAG) $(IMG_REPO):$(VERSION)
	docker push $(IMG_REPO):$(VERSION)
