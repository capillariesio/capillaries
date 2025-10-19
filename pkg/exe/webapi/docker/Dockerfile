FROM golang:1.25 as golang
WORKDIR /usr/src/capillaries
ADD go.mod go.sum ./
ADD ./pkg ./pkg
RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -o /usr/local/bin/capiwebapi /usr/src/capillaries/pkg/exe/webapi
# Capitoolbelt is optional, but useful:
# it allows users to start processes from the command line without using UI, for example:
# docker exec -it capillaries_webapi /usr/local/bin/capitoolbelt start_run -script_file=/tmp/capi_cfg/py_calc_quicktest/script.json -params_file=/tmp/capi_cfg/py_calc_quicktest/script_params.json -keyspace=py_calc_quicktest -start_nodes=read_order_items
RUN CGO_ENABLED=0 go build -o /usr/local/bin/capitoolbelt /usr/src/capillaries/pkg/exe/toolbelt

FROM scratch
COPY --from=golang /usr/local/bin/capiwebapi /usr/local/bin/ 
COPY --from=golang /usr/local/bin/capitoolbelt /usr/local/bin/ 
ADD ./pkg/exe/webapi/capiwebapi.json /usr/local/bin/
ADD ./pkg/exe/toolbelt/capitoolbelt.json /usr/local/bin/
# Scratch doesn't even have CA store (required for HTTPS and S3 acess)
COPY --from=golang /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/usr/local/bin/capiwebapi"]