# Capillaries-UI

Capillaries-UI is a simple [SPA](https://en.wikipedia.org/wiki/Single-page_application) that provides user access to Capillaries environment (message queue and Cassandra storage) using [Webapi](#webapi)

It uses [Svelte](https://svelte.dev/) for building UI components. Serves at 8080 in dev mode. In production environment, consider using some production-grade web server.

## Requirements

Node.js + npm

## Building

Get all dependencies (used in Capillaries UI [container](docker/Dockerfile)):

```
npm install
```

Build static bundle:

```
npm run build (used in Capillaries UI [container](docker/Dockerfile)):
```

Build and serve in dev mode:

```
npm run dev
```

## Settings

### CAPI_WEBAPI_URL

Environment variable, specifies URL of [Webapi](../doc/glossary.md#webapi) to use. Default: `http:\\localhost:6543`.
