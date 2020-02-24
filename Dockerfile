FROM	golang:1.13-alpine as builder

WORKDIR	/data/

ENV	CGO_ENABLED	0
ENV	GO111MODULE	on
ENV	GOOS		linux

RUN	set -ex; \
	apk update; \
	apk add --no-cache git
RUN	mkdir -p	/data/bin/ /data/test/

COPY	go.mod	.
COPY	go.sum	.
COPY	cmd		cmd
RUN	go mod download

RUN	go build -o /data/bin/alignak /data/cmd/alignak

FROM alpine:latest

WORKDIR	/data
RUN 	apk add --no-cache tzdata
COPY --from=builder	/data/bin/	.
