FROM golang:1.25 AS golang
WORKDIR /usr/src/capillaries
ADD go.mod go.sum ./
ADD ./pkg ./pkg
RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -o /usr/local/bin/capimq /usr/src/capillaries/pkg/exe/mq

FROM scratch
COPY --from=golang /usr/local/bin/capimq /usr/local/bin/ 
ADD ./pkg/exe/mq/capimq.json /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/capimq"]