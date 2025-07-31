package docker

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"go.yaml.in/yaml/v3"

	"github.com/guessi/docker-parallel-pull/internal/config"
	"github.com/guessi/docker-parallel-pull/internal/output"
	"github.com/guessi/docker-parallel-pull/internal/progress"
	"github.com/guessi/docker-parallel-pull/internal/security"
	dockertypes "github.com/guessi/docker-parallel-pull/internal/types"
)

// CreateDockerClient creates a new Docker client with API version negotiation
func CreateDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("error creating Docker client: %w", err)
	}
	return cli, nil
}

// LoadContainerImages reads and parses the YAML file containing image names with security validation
func LoadContainerImages(filename string) ([]string, error) {
	data, err := security.SecureReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read container image list: %w", err)
	}

	var containerImageList dockertypes.ImageList
	if err := parseYAML(data, &containerImageList); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if len(containerImageList.Images) == 0 {
		return nil, fmt.Errorf("no images found in %s", filename)
	}

	if len(containerImageList.Images) > security.MaxImages {
		return nil, fmt.Errorf("too many images (%d), maximum allowed: %d", len(containerImageList.Images), security.MaxImages)
	}

	validatedImages := make([]string, 0, len(containerImageList.Images))
	for i, imageName := range containerImageList.Images {
		if err := security.ValidateImageName(imageName); err != nil {
			return nil, fmt.Errorf("invalid image name at index %d: %w", i, err)
		}
		validatedImages = append(validatedImages, imageName)
	}

	return validatedImages, nil
}

// calculateBackoffDelay calculates exponential backoff delay
func calculateBackoffDelay(attempt int, baseDelay time.Duration) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}

	multiplier := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(baseDelay) * multiplier)

	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// pullImageWithRetry pulls a single Docker image with retry logic and security validation
func pullImageWithRetry(ctx context.Context, client *client.Client, imageName string, config *config.Config) dockertypes.PullResult {
	startTime := time.Now()
	var lastErr error

	if client == nil {
		return dockertypes.PullResult{
			Image:    imageName,
			Success:  false,
			Error:    "Docker client is nil",
			Duration: time.Since(startTime),
			Attempts: 1,
		}
	}

	if config == nil {
		return dockertypes.PullResult{
			Image:    imageName,
			Success:  false,
			Error:    "Config is nil",
			Duration: time.Since(startTime),
			Attempts: 1,
		}
	}

	if err := security.ValidateImageName(imageName); err != nil {
		return dockertypes.PullResult{
			Image:    imageName,
			Success:  false,
			Error:    security.SanitizeErrorMessage(err),
			Duration: time.Since(startTime),
			Attempts: 1,
		}
	}

	for attempt := 1; attempt <= config.MaxRetries+1; attempt++ {
		pullCtx, cancel := context.WithTimeout(ctx, config.Timeout)

		r, err := client.ImagePull(pullCtx, imageName, image.PullOptions{})
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed to pull image %s: %w", attempt, security.SanitizeLogMessage(imageName), err)
			cancel()

			if attempt <= config.MaxRetries {
				delay := calculateBackoffDelay(attempt, config.RetryDelay)
				output.SecureLogMessage(config, "WARN", fmt.Sprintf("Pull failed for %s (attempt %d/%d), retrying in %v",
					security.SanitizeLogMessage(imageName), attempt, config.MaxRetries+1, delay))
				time.Sleep(delay)
				continue
			}
			break
		}

		var size int64
		var imageData []byte

		if config.ShowPullDetail {
			output.SecureLogMessage(config, "INFO", fmt.Sprintf("=== Pulling %s (attempt %d) ===", security.SanitizeLogMessage(imageName), attempt))

			limitedReader := io.LimitReader(r, security.MaxFileSize)
			data, err := io.ReadAll(limitedReader)
			if err != nil {
				r.Close()
				cancel()
				lastErr = fmt.Errorf("failed to read pull output for %s: %w", security.SanitizeLogMessage(imageName), err)
				continue
			}
			size = int64(len(data))
			imageData = data

			output.SecureLogMessage(config, "INFO", fmt.Sprintf("=== Completed %s ===", security.SanitizeLogMessage(imageName)))
		} else {
			limitedReader := io.LimitReader(r, security.MaxFileSize)
			written, err := io.Copy(io.Discard, limitedReader)
			if err != nil {
				r.Close()
				cancel()
				lastErr = fmt.Errorf("failed to complete pull for %s: %w", security.SanitizeLogMessage(imageName), err)
				continue
			}
			size = written
		}

		r.Close()
		cancel()

		var imageHash string
		if len(imageData) > 0 {
			imageHash = security.CalculateImageHash(imageData)
		}

		return dockertypes.PullResult{
			Image:     imageName,
			Success:   true,
			Duration:  time.Since(startTime),
			Attempts:  attempt,
			Size:      size,
			ImageHash: imageHash,
		}
	}

	return dockertypes.PullResult{
		Image:    imageName,
		Success:  false,
		Error:    security.SanitizeErrorMessage(lastErr),
		Duration: time.Since(startTime),
		Attempts: config.MaxRetries + 1,
	}
}

// PullImages orchestrates parallel pulling of multiple images with concurrency control
func PullImages(ctx context.Context, client *client.Client, images []string, config *config.Config) []dockertypes.PullResult {
	if client == nil || config == nil {
		return []dockertypes.PullResult{}
	}

	tracker := &progress.ProgressTracker{}
	tracker.SetTotal(int64(len(images)))

	semaphore := make(chan struct{}, config.MaxConcurrency)
	results := make(chan dockertypes.PullResult, len(images))
	var wg sync.WaitGroup

	var progressDone chan struct{}
	if config.ShowProgress && config.OutputFormat == "text" {
		progressDone = make(chan struct{})
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					progress.UpdateProgress(config, tracker)
				case <-progressDone:
					progress.UpdateProgress(config, tracker)
					return
				}
			}
		}()
	}

	for _, img := range images {
		wg.Add(1)
		go func(imageName string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			output.SecureLogMessage(config, "INFO", fmt.Sprintf("Starting pull for: %s", security.SanitizeLogMessage(imageName)))
			result := pullImageWithRetry(ctx, client, imageName, config)

			if result.Success {
				output.SecureLogMessage(config, "INFO", fmt.Sprintf("✅ Successfully pulled: %s (took %v, %d bytes)",
					security.SanitizeLogMessage(imageName), result.Duration.Round(time.Second), result.Size))
			} else {
				output.SecureLogMessage(config, "ERROR", fmt.Sprintf("❌ Failed to pull %s after %d attempts",
					security.SanitizeLogMessage(imageName), result.Attempts))
			}

			tracker.Increment(result.Success)
			results <- result
		}(img)
	}

	go func() {
		wg.Wait()
		close(results)
		if progressDone != nil {
			close(progressDone)
		}
	}()

	var pullResults []dockertypes.PullResult
	for result := range results {
		pullResults = append(pullResults, result)
	}

	return pullResults
}

// CleanupImages removes all pulled images from the local Docker registry
func CleanupImages(ctx context.Context, client *client.Client, images []string, config *config.Config) {
	if client == nil || config == nil {
		return
	}

	output.SecureLogMessage(config, "INFO", "Cleaning up pulled images...")

	removeOptions := image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}

	for _, img := range images {
		if _, err := client.ImageRemove(ctx, img, removeOptions); err != nil {
			if !strings.Contains(err.Error(), "No such image:") {
				output.SecureLogMessage(config, "WARN", fmt.Sprintf("Failed to remove image %s", security.SanitizeLogMessage(img)))
			}
		} else {
			output.SecureLogMessage(config, "INFO", fmt.Sprintf("🗑️  Removed: %s", security.SanitizeLogMessage(img)))
		}
	}
}

// parseYAML is a helper function to parse YAML with security settings
func parseYAML(data []byte, v interface{}) error {
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // Reject unknown fields for security
	return decoder.Decode(v)
}
