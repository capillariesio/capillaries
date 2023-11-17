FROM node:18.12.1-alpine

WORKDIR /usr/src/capillaries

# Capillaries daemon sources
COPY ./ui/public ./public
COPY ./ui/src ./src
COPY ./ui/*.json ./
COPY ./ui/*.js ./

# Build ui
RUN npm install
RUN npm run build

COPY ./ui/docker/docker-startup.sh /usr/src/capillaries/
RUN chmod +x /usr/src/capillaries/docker-startup.sh

# This will run "npm run start"
ENTRYPOINT ["/usr/src/capillaries/docker-startup.sh"]