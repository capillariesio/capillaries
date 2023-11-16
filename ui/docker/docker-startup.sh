#!/bin/sh

cd /usr/src/capillaries
npm run start --host >> /tmp/capi_out/capiui-$(date +"%Y%m%d%H%M").log 2>&1