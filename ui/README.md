# Capillaries-UI

Capillaries-UI is a simple [SPA](https://en.wikipedia.org/wiki/Single-page_application) that provides user access to Capillaries environment (RabbitMQ queues and Cassandra storage) using [Webapi](#webapi)

It uses [Svelte](https://svelte.dev/) for building UI components, [rollup.js](https://rollupjs.org/) for bundling, [sirv](https://www.npmjs.com/package/sirv) for serving static content.

Serves at 8080. In production environment, consider using some production-grade web server instead of sirv.

## Requirements

[Node.js + npm](https://www.npmjs.com/)

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

### CAPILLARIES_WEBAPI_URL

Environment variable, specifies URL of [Webapi](../doc/glossary.md#webapi) to use. Default: `http:\\localhost:6543`.
