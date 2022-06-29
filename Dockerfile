FROM scratch
LABEL maintainer="Red Hat Inc."
ARG GOARCH
COPY bin/amd64/kove-extended-resource-controller /kove-extended-resource-controller
ENTRYPOINT ["/kove-extended-resource-controller"]
