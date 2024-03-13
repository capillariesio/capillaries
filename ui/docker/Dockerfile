FROM node:21.7.0-alpine

WORKDIR /usr/src/capillaries

# Capillaries UI sources
COPY ./ui/static ./static
COPY ./ui/src ./src
COPY ./ui/*.json ./
COPY ./ui/*.js ./

# Build ui
RUN npm install
RUN npm run build

RUN apk update
RUN apk add nginx

COPY ./ui/docker/nginx.conf /etc/nginx
CMD ["nginx", "-g", "daemon off;"]