FROM golang:1.27.1 AS golang
WORKDIR /usr/src/capillaries
ADD go.mod go.sum ./
ADD ./pkg ./pkg
RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -o /usr/local/bin/capimq -ldflags="-s -w -X main.version=docker-$(date -u +'%Y-%m-%dT%H:%M:%SZ')" /usr/src/capillaries/pkg/exe/mq

FROM scratch
COPY --from=golang /usr/local/bin/capimq /usr/local/bin/ 
ADD ./pkg/exe/mq/capimq.json /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/capimq"]