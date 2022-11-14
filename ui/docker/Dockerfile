FROM node:18.12.1-alpine

WORKDIR /usr/src/capillaries

# Capillaries daemon sources
COPY ./ui/public ./public
COPY ./ui/src ./src
COPY ./ui/*.json ./
COPY ./ui/*.js ./

# We use startup script that replaces some env_config.json setting with supplied env variables
#COPY ./ui/docker/docker-startup.sh /usr/local/bin/
#RUN chmod +x /usr/local/bin/docker-startup.sh

# Build ui
RUN npm install
RUN npm run build

# This will run "npm run start"
ENTRYPOINT ["npm", "run", "start", "--host"]

