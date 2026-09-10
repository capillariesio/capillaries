#!/bin/bash

echo Pseudo-unit tests using gocqlmem and calling ProcessDataBatchMsg. See test_api.go comments for details. Take a while to run.

test(){
	local test_name=$1
    echo $test_name
	mkdir -p /var/tmp/capi_test/$test_name
	rm -fR /var/tmp/capi_test/$test_name/*
	go test -cover ./... -run $test_name -args -test.gocoverdir="/var/tmp/capi_test/$test_name"  | grep "/api"
}

test TestTableDoesNotExistLookup
test TestOperationTimedOutLookup
test TestDataSeriousErrorLookup
test TestIdxSeriousErrorLookup
test TestDataNotAppliedLookup
test TestIdxNotAppliedSamePresentFirstRetryLookup
test TestIdxNotAppliedSamePresentSecondRetryLookup
test TestIdxNotAppliedDiffPresentLookup

test TestTableDoesNotExistFannieMae
test TestOperationTimedOutFannieMae
test TestDataSeriousErrorFannieMae
test TestIdxSeriousErrorFannieMae
test TestDataNotAppliedFannieMae
test TestIdxNotAppliedSamePresentFirstRetryFannieMae
test TestIdxNotAppliedSamePresentSecondRetryFannieMae

mkdir -p /var/tmp/capi_test/test_api_merged
rm -fR /var/tmp/capi_test/test_api_merged/*
go tool covdata merge -i=\
/var/tmp/capi_test/TestTableDoesNotExistLookup,\
/var/tmp/capi_test/TestOperationTimedOutLookup,\
/var/tmp/capi_test/TestIdxSeriousErrorLookup,\
/var/tmp/capi_test/TestDataNotAppliedLookup,\
/var/tmp/capi_test/TestIdxNotAppliedSamePresentFirstRetryLookup,\
/var/tmp/capi_test/TestIdxNotAppliedSamePresentSecondRetryLookup,\
/var/tmp/capi_test/TestIdxNotAppliedDiffPresentLookup,\
/var/tmp/capi_test/TestTableDoesNotExistFannieMae,\
/var/tmp/capi_test/TestOperationTimedOutFannieMae,\
/var/tmp/capi_test/TestDataSeriousErrorFannieMae,\
/var/tmp/capi_test/TestIdxSeriousErrorFannieMae,\
/var/tmp/capi_test/TestDataNotAppliedFannieMae,\
/var/tmp/capi_test/TestIdxNotAppliedSamePresentFirstRetryFannieMae,\
/var/tmp/capi_test/TestIdxNotAppliedSamePresentSecondRetryFannieMae \
-o=/var/tmp/capi_test/test_api_merged
go tool covdata textfmt -i=/var/tmp/capi_test/test_api_merged -o=/var/tmp/capi_test/test_api.out
go tool cover -html=/var/tmp/capi_test/test_api.out -o=/var/tmp/test_api.html

echo Coverage report: /var/tmp/test_api.html