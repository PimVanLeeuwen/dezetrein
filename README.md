# dezetrein

**[dezetrein.com](https://dezetrein.com)** ("this train") answers one question: which Dutch train am I on, and how is it doing? Tap one button and the site finds your train from your phone's location, or type the carriage number printed inside the train. Then it shows the trip: next stations, expected times, delays, platforms, cancellations, crowding and the train's carriages.

> Status: in development. The idea has been validated on real trips (the right train is among the options offered within 15 seconds in 97 to 100% of cases); the site itself is not live yet.

## How it works

- **Find the train.** The phone sends a few location fixes. The backend compares them with the live positions of all NS trains (one shared copy of the NS feed, refreshed every 10 seconds) and returns the most likely train, plus a few alternatives. It works with coarse positions too, as phones often give inside a train.
- **Confirm.** The site asks "Are you on the train to Venlo?" and offers the alternatives by destination and carriage number, the two things travellers usually know.
- **Or search by carriage number.** The carriage (trainset) number, for example 9402, is printed inside every train.
- **Show the trip.** Stops with planned and expected times, platforms, crowding and composition, refreshed while the page is open. Every answer says how old the data is.

## Privacy

Your location is used only to find your train: it is sent in the body of one request, used in memory and then dropped. It is never stored or logged. The site sets no cookies and uses no analytics or third-party trackers.

## Repository layout

| Path | What it is |
|------|------------|
| `backend/` | Go module: API server (`cmd/server`, `internal/api`), NS API client (`internal/ns`), geometry helpers (`internal/geo`), fixture recorder (`cmd/nsrecord`) |
| `backend/testdata/ns/` | Recorded NS API responses, used as test fixtures |
| `web/` | Website: React + TypeScript + Vite + Tailwind, served by nginx, which forwards `/api` to the backend |

## Development

Requirements: Go 1.27, Node 26 (`.nvmrc`), Docker.

```bash
cd backend && go vet ./... && go test -race ./...
```

```bash
cd web && npm ci && npm run lint && npm run test:run && npm run build
```

Run both locally: `go run ./cmd/server` in `backend/` (API on :8080), and `npm run dev` in `web/` (site on :5173, proxies `/api` to :8080).

Docker images:

```bash
docker build -t dezetrein-backend backend
```

```bash
docker build -t dezetrein-web web
```

The web image listens on 8080 and forwards `/api` to `BACKEND_URL` (default `http://backend:8080`).

To refresh the NS fixtures, get a free key at [apiportal.ns.nl](https://apiportal.ns.nl) (product **Ns-App**), put it in `.env` in the repo root as `NS_API_KEY=...` (see `.env.example`; never commit `.env`), then run:

```bash
cd backend && go run ./cmd/nsrecord
```

## Data and affiliation

Train data comes from the [NS API portal](https://apiportal.ns.nl). dezetrein is an independent project and is not affiliated with or endorsed by NS (Nederlandse Spoorwegen).

## Security

See [SECURITY.md](SECURITY.md). Please report vulnerabilities privately, not as a public issue.

## License

[GPL-3.0](LICENSE)
