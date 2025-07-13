package metrics

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Connection metrics
	ConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wisdom_connections_total",
			Help: "Total number of client connections.",
		},
		[]string{"status"},
	)

	// Challenge metrics
	PuzzlesSolvedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wisdom_puzzles_solved_total",
			Help: "Total number of puzzles solved successfully.",
		},
		[]string{"difficulty"},
	)

	PuzzlesFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wisdom_puzzles_failed_total",
			Help: "Total number of puzzles that failed verification.",
		},
		[]string{"difficulty"},
	)

	// Current state metrics
	CurrentDifficulty = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wisdom_current_difficulty",
			Help: "Current proof-of-work difficulty level.",
		},
	)

	ActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wisdom_active_connections",
			Help: "Number of currently active client connections.",
		},
	)

	// Performance metrics
	SolveTimeSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wisdom_solve_time_seconds",
			Help:    "Time taken to solve proof-of-work challenges.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~32s
		},
		[]string{"difficulty"},
	)

	ProcessingTimeSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wisdom_processing_time_seconds",
			Help:    "Time taken to process client requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"outcome"},
	)

	// Adaptive difficulty metrics
	DifficultyAdjustmentsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wisdom_difficulty_adjustments_total",
			Help: "Total number of difficulty adjustments.",
		},
		[]string{"direction"},
	)

	AverageSolveTimeSeconds = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wisdom_average_solve_time_seconds",
			Help: "Average solve time for recent challenges.",
		},
	)

	ConnectionRatePerMinute = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wisdom_connection_rate_per_minute",
			Help: "Current connection rate per minute.",
		},
	)
)

func init() {
	// Register all metrics
	prometheus.MustRegister(
		ConnectionsTotal,
		PuzzlesSolvedTotal,
		PuzzlesFailedTotal,
		CurrentDifficulty,
		ActiveConnections,
		SolveTimeSeconds,
		ProcessingTimeSeconds,
		DifficultyAdjustmentsTotal,
		AverageSolveTimeSeconds,
		ConnectionRatePerMinute,
	)
}

// RecordConnection tracks a new client connection
func RecordConnection(status string) {
	ConnectionsTotal.WithLabelValues(status).Inc()
}

// RecordPuzzleSolved tracks a successfully solved puzzle
func RecordPuzzleSolved(difficulty int, solveTime time.Duration) {
	diffStr := string(rune(difficulty + '0'))
	PuzzlesSolvedTotal.WithLabelValues(diffStr).Inc()
	SolveTimeSeconds.WithLabelValues(diffStr).Observe(solveTime.Seconds())
}

// RecordPuzzleFailed tracks a failed puzzle verification
func RecordPuzzleFailed(difficulty int) {
	diffStr := string(rune(difficulty + '0'))
	PuzzlesFailedTotal.WithLabelValues(diffStr).Inc()
}

// UpdateCurrentDifficulty updates the current difficulty gauge
func UpdateCurrentDifficulty(difficulty int) {
	CurrentDifficulty.Set(float64(difficulty))
}

// RecordProcessingTime tracks request processing time
func RecordProcessingTime(outcome string, duration time.Duration) {
	ProcessingTimeSeconds.WithLabelValues(outcome).Observe(duration.Seconds())
}

// RecordDifficultyAdjustment tracks difficulty changes
func RecordDifficultyAdjustment(direction string) {
	DifficultyAdjustmentsTotal.WithLabelValues(direction).Inc()
}

// UpdateStats updates aggregate statistics
func UpdateStats(avgSolveTime time.Duration, connectionRate float64, activeConns int) {
	AverageSolveTimeSeconds.Set(avgSolveTime.Seconds())
	ConnectionRatePerMinute.Set(connectionRate)
	ActiveConnections.Set(float64(activeConns))
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func StartMetricsServer(port string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(port, mux); err != nil {
			// Ignore "address already in use" errors for tests
			if !strings.Contains(err.Error(), "address already in use") {
				panic(err)
			}
		}
	}()
}
