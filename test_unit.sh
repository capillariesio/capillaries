#!/bin/bash

go test $(find ./ -name '*_test.go' -printf "%h\n" | sort -u)