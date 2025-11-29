# Go ETL Framework - Rust vs Go Performance Comparison

**Date:** 2024-11-24
**Purpose:** Extract gPRO's ETL framework to create a Go implementation for benchmarking against the existing Rust ETL implementation.

## Goal

Build a Go ETL system that:
1. Matches the Rust ETL functionality (MongoDB → PostgreSQL)
2. Uses the same test dataset (10,000 users)
3. Provides fair performance comparison between Rust and Go
4. Extracts reusable ETL patterns from gPRO

## Architecture Overview

### Four Main Components

```
┌─────────────────────────────────────────────────┐
│           ETL Manager                           │
│  - Runs multiple pipelines concurrently         │
│  - Semaphore-limited worker pool                │
│  - Graceful cancellation (context.Context)      │
└─────────────┬───────────────────────────────────┘
              │
              ├──> Pipeline 1: ETL[MongoUser, TransformedUser]
              │    └─> Extract → Bucket → Transform → Load
              │
              └──> Pipeline N: ETL[E, T]
                   └─> Extract → Bucket → Transform → Load
```

### Project Structure

```
go-etl/
├── pkg/
│   ├── etl/
│   │   ├── etl.go        # Core ETL runner (from gPRO)
│   │   └── manager.go    # Pipeline manager (gPRO + Rust patterns)
│   └── bucket/
│       └── bucket.go     # Batching system (simplified gPRO)
├── cmd/
│   └── benchmark/
│       ├── main.go       # Entry point with manager
│       ├── models_mongo.go   # MongoDB models
│       ├── models_postgres.go # PostgreSQL models (GORM)
│       └── pipeline.go   # User ETL implementation
├── go.mod
└── README.md
```

## Component Details

### 1. ETL Framework (`pkg/etl/etl.go`)

**Source:** `ext/pkg/etl/etl.go` from gPRO

**Core Interface:**
```go
type ETLProcessor[E, T any] interface {
    Extract(ctx context.Context) (<-chan E, error)
    Transform(ctx context.Context, e E) T
    Load(ctx context.Context, data []T) error
    PreProcess(ctx context.Context) error
    PostProcess(ctx context.Context) error
}
```

**Runner:**
```go
type ETL[E, T any] struct {
    processor ETLProcessor[E, T]
}

func (e *ETL[E, T]) Run(ctx context.Context, bucketCfg *BucketConfig) error {
    // 1. PreProcess
    // 2. Extract -> channel
    // 3. Bucket batching
    // 4. Transform + Load in batches
    // 5. PostProcess
}
```

**Changes from gPRO:**
- ✅ Keep: Generic ETL pattern, channel-based extraction
- ❌ Remove: Titan context (use standard context.Context)
- ❌ Remove: Multi-tenancy logic (careProviderId)
- ❌ Remove: Incremental sync tracking (lastSync)

### 2. Pipeline Manager (`pkg/etl/manager.go`)

**Source:** Adapted from:
- gPRO: `ares/service/etl/builder/base.go` (ETLManager pattern)
- Rust: `etl-rust/src/etl/manager.rs` (concurrency model)

**Manager:**
```go
type ETLManager struct {
    pipelines    []ETLRunner
    workerNum    int              // Max concurrent pipelines
    bucketConfig *BucketConfig
}

type ETLRunner interface {
    Name() string
    Run(ctx context.Context, cfg *BucketConfig) error
}

func (m *ETLManager) RunAll(ctx context.Context) error {
    // Semaphore-limited parallel execution
    // Channel-based result collection
    // Error aggregation
}
```

**Concurrency Model:**
- Semaphore limits concurrent pipelines (default: 4)
- Each pipeline runs independently
- Errors from any pipeline fail the entire run
- Graceful shutdown via context cancellation

### 3. Bucket System (`pkg/bucket/bucket.go`)

**Source:** Simplified from gPRO `ext/pkg/bucket/`

**Configuration:**
```go
type BucketConfig struct {
    BatchSize  int           // Items per batch (500)
    WorkerNum  int           // Parallel workers (runtime.NumCPU() * 2)
    Timeout    time.Duration // Flush timeout (5s)
}
```

**Operation:**
1. Consume items from channel
2. Batch into groups of `BatchSize`
3. Flush on timeout or when full
4. Process batches in parallel with `WorkerNum` workers
5. Apply backpressure when workers are saturated

### 4. MongoDB → PostgreSQL Implementation

**MongoDB Models** (matching Rust exactly):
```go
type MongoUser struct {
    ID           int64       `bson:"_id"`
    Username     string      `bson:"username"`
    Email        string      `bson:"email"`
    FirstName    string      `bson:"first_name"`
    LastName     string      `bson:"last_name"`
    Age          int32       `bson:"age"`
    CreatedAt    time.Time   `bson:"created_at"`
    UpdatedAt    time.Time   `bson:"updated_at"`
    Address      Address     `bson:"address"`         // Nested
    Profile      Profile     `bson:"profile"`         // Nested
    Preferences  Preferences `bson:"preferences"`     // Nested
    ActivityLog  []LogEntry  `bson:"activity_log"`    // Array
    Transactions []Transaction `bson:"transactions"`  // Array
    Messages     []Message   `bson:"messages"`        // Array
    SocialMedia  SocialMedia `bson:"social_media"`    // Nested
    LargeData    LargeData   `bson:"large_data"`      // Nested
}
```

**PostgreSQL Models** (15 tables, same as Rust):
```go
// Using GORM (Go's equivalent of SeaORM)
type User struct {
    ID        int64     `gorm:"primaryKey"`
    Username  string    `gorm:"type:varchar(255);unique;not null"`
    Email     string    `gorm:"type:varchar(255);unique;not null"`
    FirstName string    `gorm:"type:varchar(255);not null"`
    LastName  string    `gorm:"type:varchar(255);not null"`
    Age       int32
    CreatedAt time.Time `gorm:"not null"`
    UpdatedAt time.Time `gorm:"not null"`
}

// Additional 14 tables:
// - addresses, profiles, education, experience
// - preferences, settings, activity_log, transactions
// - messages, attachments, social_media, posts
// - groups, large_data
```

**Transform Logic:**
- Flatten nested MongoDB documents → relational tables
- Generate IDs for array elements: `userId * 10000 + index`
- Convert BSON types to Go types
- Handle JSON fields (coordinates, interests, skills)

**Load Strategy:**
```go
func (p *UserETL) Load(ctx context.Context, items []TransformedUser) error {
    // 1. Collect all entities by table
    users := []User{}
    addresses := []Address{}
    // ... for all 15 tables

    // 2. Batch insert in dependency order
    db.CreateInBatches(users, 500)
    db.CreateInBatches(addresses, 500)
    db.CreateInBatches(profiles, 500)
    // ... etc

    return nil
}
```

## Performance Metrics

### Configuration (matching Rust)

```go
const (
    DatasetSize = 10_000  // Same MongoDB dataset
    BatchSize   = 500     // Same as Rust
    WorkerNum   = runtime.NumCPU() * 2  // Same as Rust
    ManagerWorkers = 1    // Same as Rust
)
```

### Metrics Collection

```go
type Metrics struct {
    TotalUsers       int64
    TotalRecords     int64         // Across all 15 tables
    Duration         time.Duration
    UsersPerSecond   float64
    RecordsPerSecond float64
}
```

### Profiling

**CPU Profiling:**
```go
import _ "net/http/pprof"

f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()
```

**Memory Profiling:**
```go
runtime.MemProfileRate = 1
f, _ := os.Create("mem.prof")
pprof.WriteHeapProfile(f)
```

### Output Format (matching Rust)

```
ETL with Database Example
=========================

Connecting to databases...
Connected successfully!

Running PostgreSQL migrations...
✓ Migrations completed successfully!

✓ Adding User ETL Pipeline (MongoDB -> PostgreSQL)

--- Starting ETL pipeline ---

Batch inserting 500 users...
Batch inserting 500 addresses...
✓ Batch inserted 500 users with all related data!
...

=== ETL process completed successfully in X.XXs ===

Performance Metrics:
- Total Users: 10,000
- Total Records: ~345,000
- Duration: X.XXs
- Throughput: X,XXX users/second
- Record Rate: XX,XXX records/second
```

## Comparison Framework

### Benchmark Execution

```bash
# Run Rust benchmark
cd etl-rust/example
cargo run --release

# Run Go benchmark
cd go-etl
go run cmd/benchmark/main.go
```

### Comparison Report

```
╔═══════════════════════════════════════╗
║      Rust vs Go Performance           ║
╠═══════════════════════════════════════╣
║ Rust:    2.89s  (~3,460 users/s)     ║
║ Go:      X.XXs  (~X,XXX users/s)     ║
║ Winner:  [Rust/Go] by XX%            ║
╚═══════════════════════════════════════╝

Detailed Comparison:
┌─────────────────┬────────┬────────┬──────────┐
│ Metric          │ Rust   │ Go     │ Diff %   │
├─────────────────┼────────┼────────┼──────────┤
│ Users/sec       │ 3,460  │ X,XXX  │ ±XX%     │
│ Records/sec     │119,000 │XX,XXX  │ ±XX%     │
│ Memory (MB)     │ XXX    │ XXX    │ ±XX%     │
│ CPU (cores)     │ X.XX   │ X.XX   │ ±XX%     │
└─────────────────┴────────┴────────┴──────────┘
```

## Implementation Plan

### Phase 1: Extract Framework
1. Extract `pkg/etl/etl.go` from gPRO
2. Remove Titan dependencies
3. Add generic types support
4. Unit tests for ETL runner

### Phase 2: Build Manager
1. Extract manager pattern from gPRO
2. Add Rust-style concurrency (semaphore + channels)
3. Context-based cancellation
4. Integration tests

### Phase 3: Implement Bucket
1. Simplify gPRO bucket for standalone use
2. Configurable batching + timeout
3. Worker pool implementation
4. Backpressure handling

### Phase 4: MongoDB → PostgreSQL
1. Define MongoDB models (match Rust)
2. Define PostgreSQL models (GORM)
3. Implement Extract (MongoDB cursor)
4. Implement Transform (flatten nested docs)
5. Implement Load (batch inserts)
6. GORM migrations (15 tables)

### Phase 5: Metrics & Profiling
1. Add duration tracking
2. Add throughput calculation
3. CPU profiling integration
4. Memory profiling integration
5. Comparison report generator

### Phase 6: Testing & Benchmarking
1. Run with 10K dataset
2. Collect metrics
3. Compare with Rust results
4. Analyze performance differences
5. Document findings

## Success Criteria

- ✅ Go ETL processes same 10K dataset as Rust
- ✅ Same 15-table schema structure
- ✅ Comparable or better performance metrics
- ✅ Clean, reusable ETL framework extracted from gPRO
- ✅ Comprehensive comparison report
- ✅ Code profiling data for analysis

## Dependencies

```go.mod
module go-etl

go 1.21

require (
    go.mongodb.org/mongo-driver v1.13.0
    gorm.io/gorm v1.25.5
    gorm.io/driver/postgres v1.5.4
    github.com/joho/godotenv v1.5.1
)
```

## Configuration

**Environment Variables:**
```bash
MONGODB_URL=mongodb://admin:password123@localhost:27017/
POSTGRES_URL=postgresql://postgres:postgres@localhost:5432/etl_example
```

**Benchmark Config:**
```go
type Config struct {
    BatchSize     int           `default:"500"`
    WorkerNum     int           `default:"runtime.NumCPU() * 2"`
    ManagerWorkers int          `default:"1"`
    Timeout       time.Duration `default:"5s"`
}
```

## Notes

- **Fair Comparison:** Same dataset, same batch sizes, same worker counts
- **Minimal Extraction:** Only essential ETL patterns from gPRO
- **No Multi-tenancy:** Simplified for benchmark focus
- **No Incremental Sync:** Full dataset migration each run
- **Clean Shutdown:** Context cancellation for graceful stops
