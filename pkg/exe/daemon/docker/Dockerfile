FROM golang:1.19

WORKDIR /usr/src/capillaries

# Capillaries daemon sources
COPY go.mod go.sum ./
COPY ./pkg ./pkg

# Certificate dir for tests that use github.com as cfg/in data source
COPY ./test/ca ./test/ca

RUN go mod download && go mod verify

# This image has python3 pre-installed, use python3 command for Python interpreter
COPY ./pkg/exe/daemon/env_config.json /usr/local/bin/
RUN sed -i -e 's~"python_interpreter_path":[ ]*"[a-zA-Z0-9@\.:\/\-_$ ]*"~"python_interpreter_path": "python3"~g' /usr/local/bin/env_config.json

# We use startup script that replaces some env_config.json setting with supplied env variables
COPY ./pkg/exe/daemon/docker/docker-startup.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-startup.sh

# Build daemon binary
RUN go build -v -o /usr/local/bin/daemon /usr/src/capillaries/pkg/exe/daemon

# Start the startup scrip, it will tweak env_config.json and run the daemon
ENTRYPOINT ["/usr/local/bin/docker-startup.sh"]

