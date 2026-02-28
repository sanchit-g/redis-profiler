# redis-profiler

A fast, concurrent, read-only Redis keyspace memory profiler written in Go.

## What It Does

Scans your Redis keyspace and produces a memory breakdown by key pattern,
TTL health analysis, and expiry cliff detection.

**Production safe:**
- Uses `SCAN` — never `KEYS *`
- Read-only commands only
- Bounded concurrency — configurable worker count
- Uses `MEMORY USAGE key SAMPLES 5` — never blocks Redis main thread

## Installation
```bash
git clone https://github.com/yourusername/redis-profiler
cd redis-profiler
go build -o redis-profiler .
```

## Usage
```bash
# run with defaults (connects to localhost:6379)
./redis-profiler profile

# run with a config file
./redis-profiler profile --config config.yaml
```

## Configuration

Create a `config.yaml` in your working directory:
```yaml
redis:
  address: localhost:6379
  password: ""
  db: 0

scanner:
  batch_size: 100
  workers: 20

groups:
  - name: "User Sessions"
    pattern: "session:*"
  - name: "Product Cache"
    pattern: "product:cache:*"
  - name: "Rate Limiters"
    pattern: "ratelimit:*"

output:
  format: table
  sort_by: memory
  cliff_threshold: 20
  cliff_window_minutes: 5
```

## Status

Work in progress. Report renderer and expiry cliff detection coming next.
