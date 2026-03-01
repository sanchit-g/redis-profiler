# redis-profiler

A fast, concurrent, read-only Redis keyspace memory profiler written in Go.

## Overview

`redis-profiler` is a command-line tool designed to analyze your Redis keyspace without impacting production performance. It groups keys by configurable patterns, calculates aggregate memory consumption using `MEMORY USAGE`, tracks TTL (Time To Live) health, and detects upcoming expiry cliffs that could cause cache stampedes.

**Production Safe Design:**
* **`SCAN` based:** Uses cursor-based iteration instead of the blocking `KEYS *` command.
* **Read-only:** Executes strictly read-only commands against the keyspace.
* **Bounded Concurrency:** Utilizes a configurable worker pool to prevent overloading the Redis server.
* **Non-blocking Memory Analysis:** Automatically utilizes `MEMORY USAGE key SAMPLES 5` to ensure the Redis main thread is never blocked during analysis.

## Features

* **Memory Breakdown:** Aggregates total keys, average key size, and overall memory footprint by logical groups.
* **TTL Analysis:** Identifies the percentage of keys missing a TTL (Time To Live) to highlight potential memory leaks.
* **Expiry Cliff Detection:** Warns if a high percentage of keys within a group are scheduled to expire simultaneously within a short time window.
* **High Performance:** Go-based concurrent pipeline utilizing bounded workers.
* **Flexible Configuration:** Supports YAML configuration files and environment variable overrides.

## Installation

### Prerequisites
* Go 1.20 or higher

### Build from Source
```bash
git clone https://github.com/sanchit-g/redis-profiler
cd redis-profiler
go build -o redis-profiler main.go
```

## Usage

Run the profiler using the default settings (connects to `localhost:6379`):
```bash
./redis-profiler profile
```

Run with a specific configuration file:
```bash
./redis-profiler profile --config config.yaml
```

## Configuration

The application is configured via a `config.yaml` file. You can customize the Redis connection, scanner behavior, key groupings, and output formatting.

### Example `config.yaml`

```yaml
redis:
  address: "localhost:6379"
  password: ""
  db: 0
  tls: false

scanner:
  batch_size: 100       # Number of keys fetched per SCAN iteration
  workers: 20           # Number of concurrent goroutines calling MEMORY USAGE

groups:
  - name: "User Sessions"
    pattern: "session:*"
  - name: "Product Cache"
    pattern: "product:cache:*"
  - name: "Rate Limiters"
    pattern: "ratelimit:*"

output:
  format: "table"       # Output format (table)
  sort_by: "memory"     # Sorting criteria
  cliff_threshold: 20   # Warn if >20% of keys expire in the same window
  cliff_window_minutes: 5 # Size of the expiry window buckets
  cliff_ignore_groups:
    - "Rate Limiters"   # Groups to exclude from cliff detection
```

### Environment Variables

All configuration options can be overridden using environment variables prefixed with `REDISPROFILER_`. For example:

* `REDISPROFILER_REDIS_ADDRESS=redis.production.local:6379`
* `REDISPROFILER_REDIS_PASSWORD=secret`
* `REDISPROFILER_SCANNER_WORKERS=50`

## Output Example

```text
Redis Memory Profiler
Connected: localhost:6379  |  Total keys: 1,500,000  |  Total memory: 1.45 GB

GROUP                  KEYS         MEMORY         AVG SIZE     NO TTL  
────────────────────────────────────────────────────────────────────────
User Sessions          1000000      953.67 MB      1000 B       10%
Product Cache          500000       500.00 MB      1048 B       0%
Rate Limiters          0            0 B            0 B          0%
────────────────────────────────────────────────────────────────────────
TOTAL                  1500000      1.45 GB       

⚠  BULK EXPIRY EVENTS DETECTED

   Group   : Product Cache
   Window  : 300s – 600s from now  |  400000 keys (80.00% of group)
   Note    : If these keys receive read traffic at expiry,
             cache miss rates may spike.
```

## Testing

The project is highly tested. You can run the test suite locally:

```bash
go test -v ./...
```
