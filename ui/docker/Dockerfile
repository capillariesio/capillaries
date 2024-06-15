FROM node:21.7.0-alpine

WORKDIR /usr/src/capillaries

# Capillaries UI sources
ADD ./ui/static ./static
ADD ./ui/src ./src
ADD ./ui/*.json ./
ADD ./ui/*.js ./
ADD ./ui/*.env ./

# Build ui
RUN npm install

# Run ui
ENTRYPOINT npm run dev