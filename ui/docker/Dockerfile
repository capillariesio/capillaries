FROM node:21.7.0-alpine

WORKDIR /usr/src/capillaries

# Capillaries UI sources
COPY ./ui/static ./static
COPY ./ui/src ./src
COPY ./ui/*.json ./
COPY ./ui/*.js ./
COPY ./ui/*.env ./

# Build ui
RUN npm install

# Run ui
ENTRYPOINT npm run dev --host > /tmp/capi_out/ui-$(date +"%Y%m%d%H%M").log 2>&1