#!/bin/bash

export PATH=$PATH:$HOME/.nvm/versions/node/v18.12.1/bin

# Just build for http://localhost:6543, sh/webapi/config.sh will patch bundle.js on deployment
# If, for some reason, patching is not an option, build UI with this env variable set:
# export CAPILLARIES_WEBAPI_URL=http://$EXTERNAL_IP_ADDRESS:6543

cd ../../../../ui
npm run build