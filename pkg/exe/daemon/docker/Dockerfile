FROM golang:1.22 as build
WORKDIR /usr/src/capillaries
COPY go.mod go.sum ./
COPY ./pkg ./pkg
RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -o /usr/local/bin/capidaemon /usr/src/capillaries/pkg/exe/daemon

FROM python:3.7-alpine
# Portfolio test needs dateutil
RUN pip install python-dateutil
COPY --from=build /usr/local/bin/capidaemon /usr/local/bin/ 
COPY ./pkg/exe/daemon/capidaemon.json /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/capidaemon"]


