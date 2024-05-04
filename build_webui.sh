#!/bin/bash

# Just build for http://localhost:6543, sh/webapi/config.sh will patch bundle.js on deployment
# If, for some reason, patching is not an option, build UI with this env variable set:
# export CAPILLARIES_WEBAPI_URL=http://$EXTERNAL_IP_ADDRESS:6543

pushd ./ui
npm run build

pushd build
tar -czvf all.tgz *
popd



cp ./test/ca/*.pem $DIR_BUILD_CA
pushd $DIR_BUILD_CA
tar cvzf all.tgz *.pem --remove-files
popd
