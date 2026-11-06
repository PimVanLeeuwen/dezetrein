# How dezetrein finds your train

The site has to answer "which train am I on?" from a phone's location alone, often a coarse one, and from the NS live vehicle feed. This page describes how the matcher in [`backend/internal/match`](../backend/internal/match) does that, why, and how it was tuned. The numbers come from a field test on real trips; the recordings themselves are personal location data and are not in this repository.

## Inputs

**From the phone:** 1 to 5 location readings, posted together in the body of one `POST /api/match`. Each has latitude, longitude, accuracy and a timestamp; speed and heading are optional. The page collects readings for about 15 seconds and posts five spread over that time. Coarse readings are accepted as they come: inside a train, phones often give only a cell or Wi-Fi position (accuracy around 1,400 m) without speed or heading, and users will not change their location settings for a website.

**From NS:** the Virtual Train API vehicle list, about 330 trains with position, speed, heading and an accuracy field, but no timestamp. One shared poller fetches it every 10 seconds for all users and keeps the snapshots of the last 5 minutes.

## What the field test showed about the NS feed

- **Positions are old when they arrive.** NS refreshes a train's position only every 10 to 20 seconds, and each position is about 6 seconds behind reality. Positions are on the right track (about 10 m sideways) but behind the train.
- **The accuracy field encodes the age.** While a position is not refreshed, its accuracy grows by the distance the train covers, on top of a base of about 200 m (about 390 m per 10 s at 134 km/h).
- **The reported heading can be stale.** One train reported 262 degrees for minutes while running at about 330 degrees.
- **10 second polling is the right rate.** Polling faster gains little because NS itself refreshes every 10 to 20 seconds; polling every 10 s halves the load compared with 5 s.

## Method

### 1. Dead reckoning

Each NS position is pushed forward along the train's course to estimate where the train is at the moment of the reading:

```
distance = growth + speed × (6 s lag + time from snapshot to reading)
growth   = max(0, accuracy − 200 m) × min(1, speed / 80 km/h)
```

- The **course** is the bearing from the train's most recent earlier position at least 30 m away, from the snapshot history. The reported heading is only a fallback.
- The **time from snapshot to reading** can be negative (the reading came first). Snapshots are 10 s apart, and at 130 km/h a train covers 180 m in 5 s.
- Below 80 km/h the **growth is scaled down** with speed: around stations the accuracy field grows faster than a braking or starting train moves.
- Trains below 5 km/h are taken as standing and not moved. The projection is capped at 2 minutes of travel, so a position frozen on a curve does not drift far off the track.

### 2. A window of readings

- Readings within 3 minutes of the newest one count. Each is paired with the snapshot taken closest to it, within 20 seconds. If no reading has a snapshot that close, the phone's clock is assumed to be off, and the newest reading is anchored to the newest snapshot.
- Around each reading, trains within a search radius are considered: twice the reading's accuracy, at least 3 km and at most 10 km, plus 2 km because raw NS positions lag behind.
- **A train must be near every reading.** Users cannot leave their train, so a train that only passes by, or appears in the feed later, is dropped.
- Trains are ranked by their **mean corrected distance** over the readings. A train on a parallel track going the other way, or a slower sprinter alongside, falls behind within seconds.

### 3. Phone speed and heading

Only from readings accurate to 100 m or better, and only when the phone reports them. A speed difference beyond 25 km/h adds 10 m per km/h. A heading difference beyond 45 degrees adds 8 m per degree, counted only when both the phone (5 m/s or more) and the train (20 km/h or more) are moving. Each penalty is capped at 1,000 m. Phones rarely reported speed in the field test, so this has had little effect so far.

### 4. Confidence and the answer

| Confidence | When |
|------------|------|
| `high` | one train in range, or the runner-up at least 2 times as far as the best, measured from at least 150 m |
| `low` | two or more trains without a clear winner |
| `none` | no train near every reading |

The API returns up to 3 candidates, each with what travellers recognise a train by: its destination and carriage numbers. The page always confirms, even at `high`: "Are you on the train to Venlo?", then the alternatives. `high` only changes the wording and sets `match`.

## Results

The matcher was replayed on three recorded trips: one with coarse GPS (median accuracy about 1,400 m, almost no speed or heading), and two with precise GPS (median about 10 m). The replay simulates a user opening the page every 30 seconds, under the site's real constraints: one snapshot per 10 seconds and five readings per request. Openings during gaps in the recording (no reading, or no snapshot in the last 30 seconds) are left out.

| Trip | After 15 s: top 1 | Top 3 | Confident and wrong | After 60 s: top 1 | Top 3 |
|------|-------------------|-------|---------------------|-------------------|-------|
| Coarse GPS (92 openings) | 85% | 98% | 1% | 89% | 97% |
| Precise GPS (62 openings) | 89% | 98% | 0% | 99% | 100% |
| Precise GPS, short (3 openings) | 100% | 100% | 0% | 100% | 100% |

The first field analysis, with every recorded snapshot and every reading, found top 3 at 97 to 100% and a confident wrong answer in 0 to 2%. The matcher reaches that under the site's constraints; the "Are you on the train to ...?" confirmation catches the rare confident mistake.

What changed results during tuning:

| Change | Effect (top 1 after 15 s) |
|--------|---------------------------|
| Projecting to each reading's own time, not the snapshot's | confident wrong answers on a precise trip from 2% to 0% |
| Scaling the growth down below 80 km/h | coarse from 78 to 85%, precise from 82 to 89% |
| Dropping the growth term entirely | better on one precise trip, worse with coarse GPS, and breaks the frozen-position case: rejected |
| Phone speed and heading penalty, projection cap | no measurable change |
| A snapshot every 5 s instead of every 10 s | about 7 points more top 1 on a precise trip, at double the NS load: not taken |

## Carriage numbers

The carriage (trainset) number, for example 9402, is printed inside every train, and travellers know it. NS has no lookup from carriage to train: the link exists only per journey. `POST /api/carriage` therefore:

1. checks a **passive index** of the carriages seen in compositions and journeys the site fetched in the last 10 minutes (trip views, matches, earlier lookups), at no extra NS cost;
2. with a location, even a coarse one, checks the compositions of the **12 trains nearest** to the newest reading, through the 10-minute composition cache;
3. otherwise answers `"train": null` with `"located": false`, so the page can ask for a location or the journey number.

A background sweep of every running train's composition would find almost any carriage, but at about 2.5 times the vehicle poller's NS load; it was not taken.

## Tests

- **Synthetic scenarios** (`match_test.go`, run in CI) use a simulator of the feed's measured behaviour: 15 s refreshes, 6 s lag, accuracy growing with age, deterministic GPS noise. They cover a train passing the other way on a parallel track, a sprinter alongside, a position frozen for a minute, a stale reported heading, a busy station, a phone clock that is off, and a train that leaves the feed. The dead-reckoning scenarios fail with dead reckoning switched off.
- **Replay** (`replay_test.go`) runs the recorded trips when they are present locally and is skipped otherwise, as in CI. It fails when, for any trip with at least 10 openings, top 3 after 15 s drops below 95%, top 1 below 80%, or confident wrong answers rise above 3%.

To replay your own recordings, put them in `backend/data/` (gitignored) as JSON lines: readings as `{"session", "t" (milliseconds since 1970), "lat", "lng", "acc", "speed", "heading"}`, and vehicle snapshots as `{"t" (RFC 3339), "vehicles": [NS vehicle objects]}`. Then add a `replay.json`:

```json
{
  "fixes": "fixes.jsonl",
  "vehicles": "vehicles.jsonl",
  "segments": [
    {"name": "my trip", "session": "<session id prefix>", "train": "<journey number>", "from": "<RFC 3339>", "to": "<RFC 3339>"}
  ]
}
```

```bash
cd backend && go test -run Replay -v ./internal/match
```
