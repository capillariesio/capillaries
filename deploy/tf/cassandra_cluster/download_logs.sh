#!/bin/bash
if [ ! -d ./log ]; then
  mkdir ./log
fi

aws s3 sync s3://$CAPILLARIES_AWS_TESTBUCKET/log ./log/

for f in ./log/*.gz ; do gunzip -c "$f" > "${f%.*}" ; done

pushd ./log
find . -name "capiwebapi-*.log" -print0 | sort -z | xargs -0 cat > capiwebapi.log
find . -name "capidaemon-*.log" -print0 | sort -z | xargs -0 cat > capidaemon.log
find . -name "cassandra-*.log" -print0 | sort -z | xargs -0 cat > cassandra.log
# This will likely choke on big files
sort -k 2 -t',' capidaemon.log > capidaemon.sorted.log 
popd

# aws s3 rm s3://$CAPILLARIES_AWS_TESTBUCKET/log/ --recursive