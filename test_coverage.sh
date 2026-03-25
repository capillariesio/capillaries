#!/bin/bash

go test -coverprofile=/var/tmp/capillaries.p.tmp -cover $(find ./ -name '*_test.go' -printf "%h\n" | sort -u)
cat /var/tmp/capillaries.p.tmp | grep -v "donotcover" > /var/tmp/capillaries.p
go tool cover -html=/var/tmp/capillaries.p -o=/var/tmp/capillaries.html
echo See /var/tmp/capillaries.html
