package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type report struct {
	APIOperations         int     `json:"api_operations"`
	WorkerOperations      int     `json:"worker_operations"`
	Errors                int64   `json:"errors"`
	P50MS                 float64 `json:"p50_ms"`
	P95MS                 float64 `json:"p95_ms"`
	MaximumMS             float64 `json:"maximum_ms"`
	ConnectionBudget      int     `json:"connection_budget"`
	ConfiguredAPIConns    int     `json:"configured_api_connections"`
	ConfiguredWorkerConns int     `json:"configured_worker_connections"`
	Passed                bool    `json:"passed"`
}

func main() {
	databaseURL := flag.String(
		"database-url", os.Getenv("LOAD_DATABASE_URL"),
		"pilot database URL",
	)
	apiConcurrency := flag.Int("api-concurrency", 8, "concurrent OLTP readers")
	workerConcurrency := flag.Int(
		"worker-concurrency", 8, "concurrent worker readers",
	)
	operations := flag.Int("operations", 50, "operations per goroutine")
	budget := flag.Int("connection-budget", 20, "total PostgreSQL budget")
	maxP95 := flag.Duration("max-p95", 500*time.Millisecond, "p95 gate")
	flag.Parse()
	if *databaseURL == "" || *apiConcurrency < 1 ||
		*workerConcurrency < 1 || *operations < 1 ||
		*apiConcurrency+*workerConcurrency > *budget {
		fmt.Fprintln(os.Stderr, "invalid load-check configuration")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	apiPool, err := openPool(ctx, *databaseURL, int32(*apiConcurrency))
	if err != nil {
		fatal(err)
	}
	defer apiPool.Close()
	workerPool, err := openPool(
		ctx, *databaseURL, int32(*workerConcurrency),
	)
	if err != nil {
		fatal(err)
	}
	defer workerPool.Close()

	var (
		lock      sync.Mutex
		latencies []time.Duration
		failures  atomic.Int64
		group     sync.WaitGroup
	)
	run := func(pool *pgxpool.Pool, query string, concurrency int) {
		for range concurrency {
			group.Add(1)
			go func() {
				defer group.Done()
				for range *operations {
					started := time.Now()
					var count int64
					err := pool.QueryRow(ctx, query).Scan(&count)
					elapsed := time.Since(started)
					lock.Lock()
					latencies = append(latencies, elapsed)
					lock.Unlock()
					if err != nil {
						failures.Add(1)
					}
				}
			}()
		}
	}
	run(
		apiPool,
		`SELECT count(*) FROM incidents WHERE status IN ('OPEN', 'IN_PROGRESS')`,
		*apiConcurrency,
	)
	run(
		workerPool,
		`SELECT count(*) FROM demonstration_capture_deliveries
		 WHERE status IN ('PENDING', 'PROCESSING', 'FAILED')`,
		*workerConcurrency,
	)
	group.Wait()
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})
	p50 := percentile(latencies, 0.50)
	p95 := percentile(latencies, 0.95)
	maximum := percentile(latencies, 1)
	result := report{
		APIOperations:         *apiConcurrency * *operations,
		WorkerOperations:      *workerConcurrency * *operations,
		Errors:                failures.Load(),
		P50MS:                 milliseconds(p50),
		P95MS:                 milliseconds(p95),
		MaximumMS:             milliseconds(maximum),
		ConnectionBudget:      *budget,
		ConfiguredAPIConns:    *apiConcurrency,
		ConfiguredWorkerConns: *workerConcurrency,
	}
	result.Passed = result.Errors == 0 && p95 <= *maxP95
	content, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(content))
	if !result.Passed {
		os.Exit(3)
	}
}

func milliseconds(value time.Duration) float64 {
	return float64(value) / float64(time.Millisecond)
}

func openPool(
	ctx context.Context,
	databaseURL string,
	maxConnections int32,
) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = maxConnections
	config.MinConns = 0
	config.ConnConfig.RuntimeParams["statement_timeout"] = "5s"
	config.ConnConfig.RuntimeParams["lock_timeout"] = "1s"
	return pgxpool.NewWithConfig(ctx, config)
}

func percentile(values []time.Duration, quantile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	index := int(float64(len(values)-1) * quantile)
	return values[index]
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
