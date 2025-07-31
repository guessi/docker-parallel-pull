package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/guessi/docker-parallel-pull/internal/config"
	"github.com/guessi/docker-parallel-pull/internal/docker"
	"github.com/guessi/docker-parallel-pull/internal/output"
)

func main() {
	// Check for config file argument
	configFile := "config.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	// Load configuration from file
	finalConfig, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config file: %v", err)
	}

	// Validate configuration
	if err := finalConfig.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create context with timeout
	totalTimeout := finalConfig.Timeout * time.Duration(finalConfig.MaxRetries+1) * 2
	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	// Create Docker client
	cli, err := docker.CreateDockerClient()
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// Test Docker connection
	if _, err := cli.Ping(ctx); err != nil {
		log.Fatalf("Cannot connect to Docker daemon: %v\nPlease ensure Docker is running and accessible.", err)
	}

	// Load container images from file
	images, err := docker.LoadContainerImages(finalConfig.ContainerFile)
	if err != nil {
		log.Fatalf("Failed to load container images: %v", err)
	}

	output.SecureLogMessage(finalConfig, "INFO",
		fmt.Sprintf("Found %d images to pull with max concurrency of %d",
			len(images), finalConfig.MaxConcurrency))

	// Pull images
	startTime := time.Now()
	results := docker.PullImages(ctx, cli, images, finalConfig)
	totalDuration := time.Since(startTime)

	// Calculate and output metrics
	metrics := output.CalculateMetrics(results, finalConfig, totalDuration)
	output.OutputResults(metrics, results, finalConfig)

	// Cleanup if requested
	if finalConfig.CleanupAfterTest {
		docker.CleanupImages(ctx, cli, images, finalConfig)
	}

	// Exit with error code if any pulls failed
	if metrics.FailureCount > 0 {
		os.Exit(1)
	}
}
