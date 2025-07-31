package types

import "time"

// PullResult contains the result of a single image pull operation
type PullResult struct {
	Image     string        `json:"image"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"` // String for security (no error details)
	Duration  time.Duration `json:"duration"`
	Attempts  int           `json:"attempts"`
	Size      int64         `json:"size,omitempty"`
	ImageHash string        `json:"image_hash,omitempty"`
}

// PullMetrics contains overall statistics for the pull operation
type PullMetrics struct {
	TotalImages     int           `json:"total_images"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	TotalRetries    int           `json:"total_retries"`
	Concurrency     int           `json:"concurrency"`
}

// ImageList represents the structure of the YAML configuration file
type ImageList struct {
	Images []string `yaml:"images,omitempty"`
}
