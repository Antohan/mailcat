---
description: Прогон качества — gofmt, go vet, go build, go test
agent: go-implementer
---

Run `./scripts/check.sh` — one call, prints only failing steps; the last line is
`OK: gofmt, vet, build, test`.

- `OK` → nothing else to say, do not repeat the output.
- Any `FAIL` → fix the cause and run the script again. Do not commit before it
  prints OK.

Do not substitute manual `gofmt`/`go vet`/`go build`/`go test` calls for the
script: four separate calls cost four times as many tokens for the same result.
