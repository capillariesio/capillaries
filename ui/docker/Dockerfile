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

# This will run "npm run start"
ENTRYPOINT ["npm", "run", "start", "--host"]

