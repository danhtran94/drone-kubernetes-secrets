#!/usr/bin/bash
# build linux amd64
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 GO111MODULE=on go build -v -a -tags netgo -o release/linux/amd64/drone-kubernetes-secrets ./cmd/drone-kubernetes-secrets
# build docker image
docker build -f docker/Dockerfile.linux.amd64 --platform linux/amd64 -t ghcr.io/danhtran94/drone-kubernetes-secrets:linux-amd64 .
# push docker image
docker push ghcr.io/danhtran94/drone-kubernetes-secrets:linux-amd64