# asv6 – Austrian Street Search

Lightweight in-memory fuzzy search engine and JSON API for Austrian addresses, built on the official
[Statistik Austria street directory](https://www.statistik.at/verzeichnis/strassenliste/gemplzstr.csv)
(~265k unique `PostalCode Municipality, StreetName` entries).

- Trigram index (32-bit keys), typical query latency 1–3 ms
- Handles typos, umlauts (`ä→ae`, `ß→ss`, …) and abbreviations
- Splits trailing house/stair/door numbers off the query (`Gasometer A/1 3.Stock 22b` → `street_number`)
- Index rebuild in the background with an atomic, lock-free swap (no downtime)
- Search-as-you-type web UI, generated OpenAPI 3.1 spec and Swagger UI

## Quick start

Requires Go 1.25+.

```bash
go run .
# open http://127.0.0.1:38417
```

On startup the CSV is downloaded and indexed in the background (a few seconds); until then `/search` returns `503`.

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-addr` | `127.0.0.1:38417` | Listen address |
| `-csv-url` | Statistik Austria URL | Street list CSV source |
| `-keep-sonderadressen` | `false` | Keep placeholder entries such as `Sonderadresse 1.Bezirk` |

### Faster builds

The default build is already optimized. For extra speed:

```bash
GOAMD64=v3 go build -trimpath -ldflags="-s -w" -o asv6 .
```

## API

| Endpoint | Description |
| --- | --- |
| `GET /search?q=<query>&limit=10&min_score=0.15` | Fuzzy address search |
| `POST /rebuild-index` | Re-download the CSV and rebuild the index in the background (`202`, or `409` if already running) |
| `GET /docs` | Swagger UI |
| `GET /openapi.json`, `/openapi.yaml` | Generated OpenAPI spec |
| `GET /` | Web UI |

```bash
curl -G "http://127.0.0.1:38417/search" \
  --data-urlencode "q=1110 Wien, Gasometer A/1 3.Stock 22b" -d limit=2
```

```json
{
  "results": [
    { "address": "1110 Wien, Gasometer A", "street_number": "1 3.Stock 22b", "score": 1 },
    { "address": "1110 Wien, Gasometer B", "street_number": "1 3.Stock 22b", "score": 0.9048 }
  ]
}
```

Errors use [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem details. The response header `X-Search-Time` reports the server-side search duration.

## How it works

1. **Ingestion** – every CSV row yields one record per distinct municipality (`Politische Gemeinde`, `Zustellort`)
   and street name (`Straßenname`, `Straßennamen Kurz`); postal codes are padded to 4 digits and records are
   deduplicated on their canonical form.
2. **Normalization** – lowercase, transliterate umlauts/accents, split on whitespace and punctuation.
3. **Indexing** – the canonical string is padded with spaces and split into trigrams, each packed into a `uint32`
   and mapped to a posting list of document ids.
4. **Search** – the full query and every right-to-left trim at word boundaries are evaluated in a single pass over
   the posting lists. The split with the strongest net evidence (matched minus missed trigrams) wins, the trimmed
   tail becomes `street_number`, and results are scored as
   `matching trigrams / unique trigrams in the trimmed query`.

## Development

```bash
go test ./...
# benchmark against real data
curl -o /tmp/gemplzstr.csv https://www.statistik.at/verzeichnis/strassenliste/gemplzstr.csv
ASV_CSV=/tmp/gemplzstr.csv go test ./internal/source -run '^$' -bench RealData
```

## License

Code: [MIT](LICENSE).

## Data source & attribution

Datenquelle: Statistik Austria — [data.statistik.gv.at](https://data.statistik.gv.at), licensed under
[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).

The data is not included in this repository; it is downloaded at runtime. **Changes made:** records are split per
municipality/street-name variant, text is normalized for search, postal codes are zero-padded, and duplicates and
“Sonderadresse” placeholder entries (unless `-keep-sonderadressen` is set) are removed. The attribution is also
shown in the web UI footer and the OpenAPI description.
