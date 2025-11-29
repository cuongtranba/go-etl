package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/cuong/go-etl/pkg/bucket"
	"github.com/cuong/go-etl/pkg/etl"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	fmt.Println("ETL with Database Example (Go)")
	fmt.Println("================================\n")

	// Load environment
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found, using environment variables\n")
	}

	mongoURL := os.Getenv("MONGODB_URL")
	postgresURL := os.Getenv("POSTGRES_URL")

	if mongoURL == "" || postgresURL == "" {
		fmt.Println("ERROR: MONGODB_URL and POSTGRES_URL must be set")
		os.Exit(1)
	}

	// Set up context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Connect to databases
	fmt.Println("Connecting to databases...")
	mongoClient, err := connectMongoDB(ctx, mongoURL)
	if err != nil {
		fmt.Printf("Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer mongoClient.Disconnect(ctx)

	postgresDB, err := connectPostgres(postgresURL)
	if err != nil {
		fmt.Printf("Failed to connect to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Connected successfully!\n")

	// Start CPU profiling
	cpuFile, err := os.Create("cpu.prof")
	if err != nil {
		fmt.Printf("Failed to create CPU profile: %v\n", err)
		os.Exit(1)
	}
	defer cpuFile.Close()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		fmt.Printf("Failed to start CPU profile: %v\n", err)
		os.Exit(1)
	}
	defer pprof.StopCPUProfile()

	// Create ETL processor
	userETL := NewUserETL(mongoClient, postgresDB)

	// Configure bucket (matching Rust)
	numCPUs := runtime.NumCPU()
	bucketConfig := &bucket.Config{
		BatchSize: 500,            // Same as Rust
		WorkerNum: numCPUs * 2,    // Same as Rust
		Timeout:   5 * time.Second,
	}

	// Configure manager (matching Rust)
	managerConfig := &etl.Config{
		WorkerNum: 1, // Same as Rust
	}

	// Create manager
	manager := etl.NewManager(managerConfig, bucketConfig)
	etl.AddPipelineGeneric(manager, userETL, "user_migration_pipeline")

	fmt.Printf("✓ Adding User ETL Pipeline (MongoDB -> PostgreSQL)\n")
	fmt.Printf("  - Batch Size: %d\n", bucketConfig.BatchSize)
	fmt.Printf("  - Workers: %d (CPUs: %d)\n", bucketConfig.WorkerNum, numCPUs)
	fmt.Printf("  - Manager Workers: %d\n\n", managerConfig.WorkerNum)

	fmt.Println("--- Starting ETL pipeline ---\n")

	// Run benchmark
	start := time.Now()
	err = manager.RunAll(ctx)
	duration := time.Since(start)

	// Stop CPU profiling
	pprof.StopCPUProfile()

	// Memory profiling
	runtime.GC() // Get up-to-date statistics
	memFile, err := os.Create("mem.prof")
	if err == nil {
		defer memFile.Close()
		pprof.WriteHeapProfile(memFile)
	}

	// Check result
	if err != nil {
		fmt.Printf("\n=== Error running pipeline: %v ===\n", err)
		os.Exit(1)
	}

	// Calculate metrics
	var userCount int64
	postgresDB.Table("users").Count(&userCount)

	var totalRecords int64
	tables := []string{"users", "addresses", "profiles", "education", "experience",
		"preferences", "settings", "activity_log", "transactions", "messages",
		"attachments", "social_media", "posts", "groups", "large_data"}

	for _, table := range tables {
		var count int64
		postgresDB.Table(table).Count(&count)
		totalRecords += count
	}

	usersPerSec := float64(userCount) / duration.Seconds()
	recordsPerSec := float64(totalRecords) / duration.Seconds()

	// Print results
	fmt.Printf("\n=== ETL process completed successfully in %.2fs ===\n", duration.Seconds())
	fmt.Println("\nPerformance Metrics:")
	fmt.Printf("- Total Users: %d\n", userCount)
	fmt.Printf("- Total Records: %d\n", totalRecords)
	fmt.Printf("- Duration: %.2fs\n", duration.Seconds())
	fmt.Printf("- Throughput: %.0f users/second\n", usersPerSec)
	fmt.Printf("- Record Rate: %.0f records/second\n", recordsPerSec)
	fmt.Printf("- CPU Cores Used: %d\n", numCPUs)
	fmt.Println("\n✓ CPU profile saved to: cpu.prof")
	fmt.Println("✓ Memory profile saved to: mem.prof")

	// Generate comparison report
	generateComparisonReport(userCount, totalRecords, duration)
}

func connectMongoDB(ctx context.Context, uri string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}

func connectPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func generateComparisonReport(userCount int64, totalRecords int64, duration time.Duration) {
	// Rust baseline from your example
	rustDuration := 2.89
	rustUsersPerSec := 3460.0
	rustRecordsPerSec := 119000.0

	goDuration := duration.Seconds()
	goUsersPerSec := float64(userCount) / goDuration
	goRecordsPerSec := float64(totalRecords) / goDuration

	diffPercent := ((goDuration - rustDuration) / rustDuration) * 100
	winner := "Rust"
	if goDuration < rustDuration {
		winner = "Go"
		diffPercent = -diffPercent
	}

	fmt.Println("\n╔═══════════════════════════════════════╗")
	fmt.Println("║      Rust vs Go Performance           ║")
	fmt.Println("╠═══════════════════════════════════════╣")
	fmt.Printf("║ Rust:    %.2fs  (~%.0f users/s)      ║\n", rustDuration, rustUsersPerSec)
	fmt.Printf("║ Go:      %.2fs  (~%.0f users/s)      ║\n", goDuration, goUsersPerSec)
	fmt.Printf("║ Winner:  %-6s by %.1f%%              ║\n", winner, diffPercent)
	fmt.Println("╚═══════════════════════════════════════╝")

	fmt.Println("\nDetailed Comparison:")
	fmt.Println("┌─────────────────┬──────────┬──────────┬──────────┐")
	fmt.Println("│ Metric          │ Rust     │ Go       │ Diff %   │")
	fmt.Println("├─────────────────┼──────────┼──────────┼──────────┤")
	fmt.Printf("│ Users/sec       │ %-8.0f │ %-8.0f │ %+7.1f%% │\n",
		rustUsersPerSec, goUsersPerSec, ((goUsersPerSec-rustUsersPerSec)/rustUsersPerSec)*100)
	fmt.Printf("│ Records/sec     │ %-8.0f │ %-8.0f │ %+7.1f%% │\n",
		rustRecordsPerSec, goRecordsPerSec, ((goRecordsPerSec-rustRecordsPerSec)/rustRecordsPerSec)*100)
	fmt.Printf("│ Duration (s)    │ %-8.2f │ %-8.2f │ %+7.1f%% │\n",
		rustDuration, goDuration, ((goDuration-rustDuration)/rustDuration)*100)
	fmt.Println("└─────────────────┴──────────┴──────────┴──────────┘")
}
