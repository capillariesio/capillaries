FROM golang:1.22

WORKDIR /usr/src/capillaries

# Capillaries daemon sources
COPY go.mod go.sum ./
COPY ./pkg ./pkg

# Certificate dir for tests that use github.com as cfg/in data source
COPY ./test/ca ./test/ca

RUN go mod download && go mod verify

# Build webapi and capitoolbelt binaries
# Capitoolbelt is optional, it allows users to start processes from the command line without using UI

RUN go build -v -o /usr/local/bin/capiwebapi /usr/src/capillaries/pkg/exe/webapi
RUN go build -v -o /usr/local/bin/capitoolbelt /usr/src/capillaries/pkg/exe/toolbelt

# Copy config files for both binaries
COPY ./pkg/exe/webapi/capiwebapi.json /usr/local/bin/
COPY ./pkg/exe/toolbelt/capitoolbelt.json /usr/local/bin/

#ENTRYPOINT /usr/local/bin/capiwebapi > /tmp/capi_out/capiwebapi-$(date +"%Y%m%d%H%M").log 2>&1
ENTRYPOINT /usr/local/bin/capiwebapi