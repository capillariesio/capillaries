FROM golang:1.19

WORKDIR /usr/src/capillaries

# Capillaries daemon sources
COPY go.mod go.sum ./
COPY ./pkg ./pkg

# Certificate dir for tests that use github.com as cfg/in data source
COPY ./test/ca ./test/ca

RUN go mod download && go mod verify

COPY ./pkg/exe/webapi/env_config.json /usr/local/bin/

# We use startup script that replaces some env_config.json setting with supplied env variables
COPY ./pkg/exe/webapi/docker/docker-startup.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-startup.sh

# Build webapi binary
RUN go build -v -o /usr/local/bin/webapi /usr/src/capillaries/pkg/exe/webapi

# Start the startup script, it will tweak env_config.json and run webapi
ENTRYPOINT ["/usr/local/bin/docker-startup.sh"]

