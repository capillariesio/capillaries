FROM golang:1.22

# These image lacks pip and dateutil
RUN apt-get update
RUN apt-get install -y apt-transport-https
RUN apt-get install -y python3-full python3-dateutil

WORKDIR /usr/src/capillaries

# Capillaries daemon sources
COPY go.mod go.sum ./
COPY ./pkg ./pkg

# Certificate dir for tests that use github.com as cfg/in data source
COPY ./test/ca ./test/ca

RUN go mod download && go mod verify

COPY ./pkg/exe/daemon/capidaemon.json /usr/local/bin/

# Build daemon binary
RUN go build -v -o /usr/local/bin/capidaemon /usr/src/capillaries/pkg/exe/daemon

ENTRYPOINT /usr/local/bin/capidaemon > /tmp/capi_out/capidaemon-$(date +"%Y%m%d%H%M").log 2>&1


