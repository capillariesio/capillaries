#!/bin/bash

# Do not include api tests, they are slow and cannot run in a single batch, use test_api.sh instead
#go test $(find ./ -not -name "api_test.go" -name '*_test.go' -printf "%h\n" | sort -u)

mkdir -p /var/tmp/capi_test/test_unit
rm -fR /var/tmp/capi_test/test_unit/*
go test -cover $(find ./ -not -name "api_test.go" -name '*_test.go' -printf "%h\n" | sort -u) -args -test.gocoverdir="/var/tmp/capi_test/test_unit"

go tool covdata textfmt -i=/var/tmp/capi_test/test_unit -o=/var/tmp/capi_test/test_unit.out
go tool cover -html=/var/tmp/capi_test/test_unit.out -o=/var/tmp/test_unit.html

echo Coverage report: /var/tmp/test_unit.html


# Old code for coverage, used in CI:
# go test -coverprofile=/var/tmp/capillaries.p.tmp -cover $(find ./ -name '*_test.go' -printf "%h\n" | sort -u)
# cat /var/tmp/capillaries.p.tmp | grep -v "donotcover" > /var/tmp/capillaries.p
# go tool cover -html=/var/tmp/capillaries.p -o=/var/tmp/capillaries.html
# echo See /var/tmp/capillaries.html
