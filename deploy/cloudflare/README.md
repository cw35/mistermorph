# Deploy MisterMorph to Cloudflare Containers

This folder contains a Cloudflare Worker + Container setup for `mistermorph serve`.

## Files

- `Dockerfile`: builds and runs `mistermorph` in daemon mode (`serve` on port `8787`)
- `src/index.js`: Worker entrypoint, routes requests to container instances
- `wrangler.jsonc`: Cloudflare Containers and Durable Object bindings
- `deploy.sh`: one-command deployment helper

## Prerequisites

- Docker
- Node.js + npm
- Cloudflare account
- `wrangler` authentication completed (`npx wrangler login`)

## Deploy

```bash
cd deploy/cloudflare
MISTER_MORPH_LLM_API_KEY="your-openai-key" ./deploy.sh
```

Optional env vars:

- `MISTER_MORPH_SERVER_AUTH_TOKEN`: bearer token for `mistermorph serve` (auto-generated if omitted)
- `MISTER_MORPH_TELEGRAM_BOT_TOKEN`: if you want to run Telegram mode later
- `WRANGLER_ENV`: target wrangler environment
- `SKIP_NPM_INSTALL=1`: skip `npm install`

## Routing

- Default instance: all requests
- Select instance via query param: `?instance=my-session`

Example:

```bash
curl -H "Authorization: Bearer <token>" "https://<worker-domain>/health"
```
