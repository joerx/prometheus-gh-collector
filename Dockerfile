FROM golang:1.25-alpine3.22 AS builder

ADD ./ /src
WORKDIR /src
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/prometheus-gh-collector .

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /src/bin/prometheus-gh-collector /usr/local/bin
ENTRYPOINT ["/usr/local/bin/prometheus-gh-collector"]
EXPOSE 9101
