# 🌦️ go-weather

> Observed weather, official alerts, and geomagnetic activity for Go — one stable vocabulary over WMO codes, CAP feeds and NOAA SWPC, for any `(latitude, longitude)`.

[![Go Reference](https://pkg.go.dev/badge/github.com/Bugs5382/go-weather.svg)](https://pkg.go.dev/github.com/Bugs5382/go-weather)
[![Go Report Card](https://goreportcard.com/badge/github.com/Bugs5382/go-weather)](https://goreportcard.com/report/github.com/Bugs5382/go-weather)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

`go-weather` is the sibling of [`go-astronomy`](https://github.com/Bugs5382/go-astronomy), and differs from it in one way worth stating first: **go-astronomy computes and go-weather observes.** Where the Sun is follows from a date and a coordinate, with no truth outside the arithmetic. Whether it is raining is a measurement somebody took, so it can only come from a feed.

That difference is the whole shape of the library. The core is pure — the vocabulary, and the judgement that maps a provider's values onto it — and every network call lives in an optional adapter subpackage the core does not import. A consumer that already has an observation never pulls in an HTTP client, and the core's tests run with nothing listening. There is a build gate for it: `task purity` fails if the core acquires `net/http` directly or transitively.

## 🚀 Quick start

```sh
go get github.com/Bugs5382/go-weather
```

```go
client := openmeteo.New()
obs, err := client.Current(ctx, weather.Coordinate{Lat: 40.68, Lng: -73.94})
if err != nil {
    return err
}

fmt.Println(obs.Condition)                 // CLOUDY
fmt.Println(obs.Quantities.CloudCover)     // 0.56
fmt.Println(obs.Wind.SpeedMPH)             // 4.2
fmt.Println(obs.Stale(time.Now()))         // false, until the provider's interval passes
```

## 📦 What is in it

| package | what it does | network |
| --- | --- | --- |
| `weather` | the vocabulary, and WMO extraction | none |
| `provider/openmeteo` | current conditions | Open-Meteo |
| `alert` | official warnings, unedited | none |
| `alert/nws` | United States alerts | api.weather.gov |
| `space` | geomagnetic activity | none |
| `space/swpc` | the planetary K index | NOAA SWPC |

Adapters are **per capability, not per vendor**. Open-Meteo has no alerts endpoint and NOAA SWPC does not do rain, so one provider per job is not a compromise — it is the shape the sources come in. Three feeds, three published standards, one vocabulary over the top.

## 🌥️ The vocabulary

Eight conditions: `CLEAR`, `CLOUDY`, `FOG`, `RAIN`, `RAIN_HEAVY`, `SNOW`, `SNOW_HEAVY`, `THUNDERSTORM`.

The grade is part of the condition because the providers grade it — WMO 61, 63 and 65 are slight, moderate and heavy rain, which is why a phone says "light rain" rather than "rain".

**There is no blizzard and no whiteout, deliberately.** The published definition of a blizzard requires the conditions to hold for three hours, which makes it a diagnosis applied in retrospect rather than something anybody can see happening. Heavy snow with a high wind — what a person standing outside actually sees — is already expressible as `SNOW_HEAVY` with a `Wind` beside it.

Wind sits beside the condition and never inside it: any condition can be calm or blowing hard, and a nominal speed per condition would collapse that distinction permanently.

Continuous quantities sit beside it too — cloud cover, precipitation, snowfall, visibility — because eight discrete states can only be cut between and numbers can be interpolated.

**A missing reading is not a zero.** Zero is a real value for every quantity: a clear sky, a dry hour, fog. When a provider gives no reading (Open-Meteo answers `null` where its model has no value at a place), the field stays `0` and `Quantities.Missing` says so. Ask with the accessors, which answer both at once:

```go
if v, ok := obs.Quantities.VisibilityReading(); ok {
    drawHaze(v) // a measured visibility, zero included
} else {
    // no reading: draw nothing rather than fog
}
```

`CloudCoverReading`, `PrecipitationReading`, `SnowfallReading` and `VisibilityReading` each return `(value, reported)`. The zero `Missing` means "everything reported", so a `Quantities` built by hand without it keeps its old meaning. A missing visibility never turns a sky into fog.

## 🚨 Alerts

An alert is something a public authority has *said*; a condition is something the sky is *doing*. They are separate types so a severity can never end up driving a rendering decision, and a tornado warning under a clear sky stays a clear sky with a warning beside it.

**Nothing is rephrased.** The issuer's event name, headline and link travel through untouched. A paraphrase is how a warning gets softened by accident.

**Absence is not safety.** `Report.Covered` distinguishes "this place has a feed and nothing is happening" from "we have no feed here". A failed request returns the zero value, whose `Covered` is false — so a consumer that ignores the error still renders ignorance rather than an all-clear it was never told.

**Only CAP `Actual` is reported.** NWS publishes test messages on the live wire: of 489 alerts active nationwide when this was written, one was a `Test`, with the same shape as a severe thunderstorm warning. `Exercise`, `System` and `Draft` are filtered with it.

## 🚫 What it does not do

No caching, no retries, no rate limiting, no scheduling, no storage, no forecasting.

Every observation carries the provider's own expiry where the provider states one, and `Observation.Stale` reports against it. Where a provider states none, none is claimed and nothing is ever called stale — choosing a TTL means knowing how often you poll and how wrong you can afford to be, which is the consumer's knowledge and not the library's. A library that reaches for a cache has opinions about deployment that a library has no business having.

## 🧪 Tests

Every adapter is tested against **recorded provider responses**, which is the only way a vocabulary mapping can be checked at all: the assertion is about somebody else's field names and units, and a hand-written fixture asserts only what its author already believed.

That is not a formality. Recording real feeds is what found the live NWS test message, and what caught that the SWPC K-index product returns objects rather than the array-of-arrays with a header row that several of its sibling products use.

```sh
task test      # go test ./...
task purity    # prove the core reaches no network
task lint      # gofmt, golangci-lint, yamllint, and the above
task license   # verify the MIT header on every source file
```
