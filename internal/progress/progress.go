package progress

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/guessi/docker-parallel-pull/internal/config"
)

// ProgressTracker tracks progress across all pull operations
type ProgressTracker struct {
	total     int64
	completed int64
	failed    int64
}

// SetTotal sets the total number of operations
func (pt *ProgressTracker) SetTotal(total int64) {
	if pt == nil {
		return
	}
	pt.total = total
}

// Increment increments the progress counters
func (pt *ProgressTracker) Increment(success bool) {
	if pt == nil {
		return
	}
	atomic.AddInt64(&pt.completed, 1)
	if !success {
		atomic.AddInt64(&pt.failed, 1)
	}
}

// GetProgress returns the current progress values
func (pt *ProgressTracker) GetProgress() (completed, failed, total int64) {
	if pt == nil {
		return 0, 0, 0
	}
	return atomic.LoadInt64(&pt.completed), atomic.LoadInt64(&pt.failed), pt.total
}

// UpdateProgress shows progress if enabled
func UpdateProgress(config *config.Config, tracker *ProgressTracker) {
	if config == nil || tracker == nil {
		return
	}
	if !config.ShowProgress || config.OutputFormat == "json" {
		return
	}

	completed, failed, total := tracker.GetProgress()
	successful := completed - failed
	percentage := float64(completed) / float64(total) * 100

	barWidth := 40
	filledWidth := int(float64(barWidth) * float64(completed) / float64(total))
	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

	fmt.Printf("\r[%s] %.1f%% (%d/%d) ✅ %d ❌ %d",
		bar, percentage, completed, total, successful, failed)

	if completed == total {
		fmt.Println()
	}
}
