# go-weather

> Go weather library: a provider-agnostic vocabulary for conditions, wind and official alerts, with Open-Meteo, NWS and NOAA SWPC adapters.

## Install

```bash
go get github.com/Bugs5382/go-weather
```

## Develop

```bash
task build    # go build ./...
task test     # go test ./...
task lint     # gofmt check + golangci-lint + yamllint
task license  # inject MIT headers (golic)
```

Commit discipline, AI-tell/emoji blocking, and the pre-push gofmt/vet/lint/test gate are enforced
by the governance hooks. Install them once per clone:

```bash
bash .claude/hooks/install.sh
```

## License

MIT (c) 2026 Shane
