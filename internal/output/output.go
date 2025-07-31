package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/guessi/docker-parallel-pull/internal/config"
	"github.com/guessi/docker-parallel-pull/internal/security"
	"github.com/guessi/docker-parallel-pull/internal/types"
)

// SecureLogMessage outputs log messages with sanitization
func SecureLogMessage(config *config.Config, level, message string) {
	if config == nil {
		return
	}

	sanitizedMessage := security.SanitizeLogMessage(message)

	if config.OutputFormat == "json" {
		logEntry := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     level,
			"message":   sanitizedMessage,
		}
		if data, err := json.Marshal(logEntry); err == nil {
			fmt.Println(string(data))
		}
	} else {
		fmt.Printf("[%s] %s: %s\n", time.Now().Format("15:04:05"), level, sanitizedMessage)
	}
}

// CalculateMetrics computes overall statistics from pull results
func CalculateMetrics(results []types.PullResult, config *config.Config, totalDuration time.Duration) types.PullMetrics {
	if config == nil {
		return types.PullMetrics{}
	}

	var successful, failed, totalRetries int
	var totalPullDuration time.Duration

	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
		totalRetries += result.Attempts - 1
		totalPullDuration += result.Duration
	}

	avgDuration := time.Duration(0)
	if len(results) > 0 {
		avgDuration = totalPullDuration / time.Duration(len(results))
	}

	return types.PullMetrics{
		TotalImages:     len(results),
		SuccessCount:    successful,
		FailureCount:    failed,
		TotalDuration:   totalDuration,
		AverageDuration: avgDuration,
		TotalRetries:    totalRetries,
		Concurrency:     config.MaxConcurrency,
	}
}

// OutputResults displays the final results based on output format
func OutputResults(metrics types.PullMetrics, results []types.PullResult, config *config.Config) {
	if config == nil {
		return
	}

	if config.OutputFormat == "json" {
		output := map[string]interface{}{
			"metrics": metrics,
			"results": results,
		}
		if data, err := json.MarshalIndent(output, "", "  "); err == nil {
			fmt.Println(string(data))
		}
	} else {
		fmt.Printf("\n📊 Pull Summary:\n")
		fmt.Printf("   ✅ Successful: %d\n", metrics.SuccessCount)
		fmt.Printf("   ❌ Failed: %d\n", metrics.FailureCount)
		fmt.Printf("   🔄 Total retries: %d\n", metrics.TotalRetries)
		fmt.Printf("   ⏱️  Total time: %v\n", metrics.TotalDuration.Round(time.Second))
		fmt.Printf("   📈 Average time per image: %v\n", metrics.AverageDuration.Round(time.Second))
		fmt.Printf("   🚀 Concurrency: %d\n", metrics.Concurrency)
	}
}
