# redis-profiler

> A fast, concurrent, read-only Redis keyspace memory profiler written in Go.

[![CI](https://github.com/sanchit-g/redis-profiler/actions/workflows/ci.yml/badge.svg)](https://github.com/sanchit-g/redis-profiler/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-v0.1.0--beta-orange.svg)](https://github.com/sanchit-g/redis-profiler/releases)

---

## The Problem

Redis memory consumption is invisible until it becomes a crisis. You know `40GB of 48GB` is used. You do not know what is consuming it.

Existing tools have real limitations:

| Tool | Problem |
|---|---|
| `KEYS *` | Blocks Redis main thread. Never safe in production. |
| `--bigkeys` | Reports only the single largest key per type. No pattern grouping. |
| `redis-memory-analyzer` | Single-threaded Python. Takes 15–45 minutes on 2–5M keys. |

`redis-profiler` scans your entire keyspace concurrently, groups keys by your own naming patterns, and gives you a memory breakdown in under 2 minutes — safe to run on production.

---

## Features

- **Memory breakdown by pattern** — see which key groups consume the most memory
- **TTL health analysis** — identify keys with no expiry (potential memory leaks)
- **Expiry cliff detection** — warns when a large percentage of keys expire simultaneously, which can cause cache stampedes
- **Production safe** — uses `SCAN` (never `KEYS *`), read-only commands only, bounded concurrency
- **Non-blocking** — uses `MEMORY USAGE key SAMPLES 5` to prevent blocking the Redis main thread
- **Live progress** — shows keys processed per second while scanning
- **Flexible config** — YAML config file with environment variable overrides

---

## Installation

### Prerequisites

- Go 1.22 or higher
- A running Redis instance

### Build From Source

```bash
git clone https://github.com/sanchit-g/redis-profiler
cd redis-profiler
go build -o redis-profiler .
```

### Verify

```bash
./redis-profiler --help
```

---

## Quick Start

**Run with defaults** (connects to `localhost:6379`, no config file needed):

```bash
./redis-profiler profile
```

**Run with a config file:**

```bash
./redis-profiler profile --config config.yaml
```

**Run with environment variable override:**

```bash
REDISPROFILER_REDIS_PASSWORD=secret ./redis-profiler profile --config config.yaml
```

---

## Configuration

Create a `config.yaml` in your working directory. The tool works without one using sensible defaults, but you need a config file to define your key grouping patterns.

```yaml
redis:
  address: "localhost:6379"
  password: ""         # use REDISPROFILER_REDIS_PASSWORD env var instead
  db: 0
  tls: false

scanner:
  batch_size: 100      # keys fetched per SCAN call
  workers: 20          # concurrent goroutines for MEMORY USAGE calls

groups:
  - name: "User Sessions"
    pattern: "session:*"
  - name: "Product Cache"
    pattern: "product:cache:*"
  - name: "Rate Limiters"
    pattern: "ratelimit:*"

output:
  format: "table"
  sort_by: "memory"
  cliff_threshold: 15        # flag if >15% of group expires in one window
  cliff_window_minutes: 10   # size of expiry window bucket
  cliff_ignore_groups:
    - "Rate Limiters"        # groups where bulk expiry is expected and normal
```

### All Configuration Options

| Key | Default | Description |
|---|---|---|
| `redis.address` | `localhost:6379` | Redis server address |
| `redis.password` | `""` | Redis password (prefer env var) |
| `redis.db` | `0` | Redis database number |
| `redis.tls` | `false` | Enable TLS (requires TLS-enabled Redis) |
| `scanner.batch_size` | `100` | Keys per SCAN iteration |
| `scanner.workers` | `20` | Concurrent worker goroutines |
| `output.cliff_threshold` | `20` | % of group expiring in one window to trigger warning |
| `output.cliff_window_minutes` | `5` | Expiry bucket window size in minutes |
| `output.cliff_ignore_groups` | `[]` | Groups excluded from cliff detection |

### Environment Variables

Every config value can be overridden with an environment variable prefixed `REDISPROFILER_`. Dots become underscores:

```bash
REDISPROFILER_REDIS_ADDRESS=prod-redis:6379
REDISPROFILER_REDIS_PASSWORD=secret
REDISPROFILER_SCANNER_WORKERS=50
```

**Security note:** Never put real passwords in `config.yaml` if you commit it to version control. Use the environment variable instead.

---

## Example Output

```
Scanning... 168,000 keys processed (49,500 keys/sec)
scan complete: 168,661 keys found

Redis Memory Profiler
Connected: localhost:6379  |  Total keys: 168661  |  Total memory: 19.59 MB

GROUP                  KEYS         MEMORY         AVG SIZE     NO TTL
────────────────────────────────────────────────────────────────────────
User Sessions          110301       13.83 MB       131 B        0%
Product Cache          44261        4.72 MB        111 B        0%
Rate Limiters          9241         649.76 KB      72 B         0%
[unclassified]         4858         417.48 KB      88 B         100%  ⚠
────────────────────────────────────────────────────────────────────────
TOTAL                  168661       19.59 MB

⚠  BULK EXPIRY EVENTS DETECTED

   Group   : User Sessions
   Window  : 300s – 900s from now  |  19615 keys (17.80% of group)
   Note    : If these keys receive read traffic at expiry,
             cache miss rates may spike.

✓  No further bulk expiry events detected
```

### Reading the Output

**`NO TTL` column** — percentage of keys in the group with no expiry set. A high number here (flagged with ⚠ above 50%) means keys will accumulate indefinitely unless explicitly deleted. This is a common cause of unexpected memory growth.

**`[unclassified]` group** — keys that matched no pattern in your config. A large unclassified group means you have Redis keys your application is creating that are not accounted for in your config. Worth investigating.

**Bulk Expiry Events** — when a large percentage of a group expires in a short window, all those cache misses hit your database simultaneously. This is a cache stampede. The tool flags windows where this risk exists so you can spread expiry times with jitter.

---

## How It Works

The tool runs a concurrent pipeline:

```
Scanner goroutine        →  work channel  →  N Worker goroutines
(SCAN loop, non-blocking)                    (MEMORY USAGE + TYPE + TTL
                                              pipelined per key)
                                                      ↓
                                             results channel
                                                      ↓
                                          Aggregator (main goroutine)
                                          (pattern match, deduplicate,
                                           accumulate stats)
                                                      ↓
                                               Report renderer
```

Key design decisions:

- `SCAN` with configurable `COUNT` — never `KEYS *`
- `MEMORY USAGE key SAMPLES 5` — estimates via sampling instead of full O(N) traversal
- xxhash-based deduplication — `SCAN` can return duplicate keys under keyspace mutation; we deduplicate using 64-bit hashes (16 bytes/entry) instead of full strings (60+ bytes/entry) to keep memory usage bounded
- Global worker pool — total concurrent connections = `workers` config value, regardless of cluster size
- Single-writer aggregator — no mutex needed on the stats map; the aggregator goroutine is the only writer

---

## Production Safety

This tool is safe to run on production Redis with the following caveats:

- **Read load** — each key requires 3 pipelined Redis commands. With 20 workers scanning 168k keys, this generates moderate read load. Reduce `workers` if you want lighter impact.
- **Memory** — the tool holds key metadata in memory during the scan. For very large keyspaces (50M+ keys), expect 800MB–1GB of RAM usage on the machine running the tool.
- **Scan duration** — a 168k key scan completes in ~3 seconds at 50k keys/sec. A 5M key scan takes roughly 1–2 minutes.

---

## Development

### Running Tests

```bash
# run all tests
go test ./...

# run with coverage
go test -cover ./...

# run only unit tests (skip integration tests that require Redis)
go test -short ./...

# run with race detector
go test -race ./...
```

### Integration Tests

Integration tests require a local Redis instance. Start one with Docker:

```bash
docker compose up -d
go test ./...
```

### Seeding Test Data

A seed script is included that creates realistic test data:

```bash
go run scripts/seed.go
```

This creates ~168k keys across several pattern groups including a bulk expiry cliff for testing the cliff detector.

---

## Contributing

This project uses a PR-based workflow. Direct pushes to `main` are blocked.

```bash
# create a feature branch
git checkout -b feat/your-feature

# make changes and commit
git add .
git commit -m "feat: description"

# push and open a PR on GitHub
git push origin feat/your-feature
```

Branch naming:

| Prefix | Use for |
|---|---|
| `feat/` | New features |
| `fix/` | Bug fixes |
| `docs/` | Documentation changes |
| `test/` | Adding or fixing tests |
| `chore/` | Tooling, config, maintenance |
| `refactor/` | Code changes with no behavior change |

CI must pass before merging. CI runs `go build`, `go test`, and `go vet` on every PR.

---

## Known Limitations (Beta)

- Table output only — JSON and CSV output modes are not yet implemented
- Standalone Redis only — cluster mode and Sentinel are not yet implemented  
- No timeout configuration — long-running scans cannot be time-bounded yet
- Pattern matching uses glob syntax only — regex patterns are not supported

---

## Roadmap

- [ ] JSON output mode for piping into monitoring tools
- [ ] Redis Cluster support
- [ ] Sentinel support
- [ ] Configurable scan timeout
- [ ] GoReleaser for pre-built binary distribution
- [ ] Benchmark comparison against redis-memory-analyzer

---

## License

MIT License — see [LICENSE](LICENSE) for details.