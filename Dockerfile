FROM golang:1.17.7-alpine
LABEL maintainer="Red Hat Inc."
ARG GOARCH
COPY util/kove.go /util/kove.go
COPY bin/amd64/kove-extended-resource-controller /kove-extended-resource-controller
ENTRYPOINT ["/kove-extended-resource-controller"]
