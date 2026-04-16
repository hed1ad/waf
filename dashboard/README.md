This is the dashboard frontend for the `waf` stack.

## Development

Run the development server:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) for local dashboard development, or use the root `docker-compose.yml` to run the full stack.

The dashboard reads aggregated events from the API and subscribes to the live `/api/stream` SSE endpoint.

## Environment

- `API_URL`: server-side base URL used by Next.js during SSR inside Docker
- `NEXT_PUBLIC_API_URL`: browser-visible API base URL

## Notes

The live feed uses Server-Sent Events, not WebSockets.
