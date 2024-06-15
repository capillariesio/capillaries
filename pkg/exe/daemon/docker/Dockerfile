FROM golang:1.22 as build
WORKDIR /usr/src/capillaries
ADD go.mod go.sum ./
ADD ./pkg ./pkg
RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -o /usr/local/bin/capidaemon /usr/src/capillaries/pkg/exe/daemon

FROM python:3.7-alpine
# FROM gcr.io/distroless/python3-debian12 no pip
# FROM python:3.12-slim 2 times bigger than alpine, still DNS errors
# Portfolio test needs dateutil
RUN pip install python-dateutil
COPY --from=build /usr/local/bin/capidaemon /usr/local/bin/ 
ADD ./pkg/exe/daemon/capidaemon.json /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/capidaemon"]


