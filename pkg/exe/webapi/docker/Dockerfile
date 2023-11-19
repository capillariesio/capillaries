FROM golang:1.21

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

# We use startup script that replaces some capi*.json setting with supplied env variables
COPY ./pkg/exe/webapi/docker/docker-startup.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-startup.sh

# Start the startup script, it will tweak capi*.json and run webapi
ENTRYPOINT ["/usr/local/bin/docker-startup.sh"]

