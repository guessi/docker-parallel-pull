package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"go.yaml.in/yaml/v3"
)

var (
	cleanupAfterTest = true
	showPullDetail   = true

	imagePullOptions   = image.PullOptions{}
	imageRemoveOptions = image.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}
)

type ImageList struct {
	Images []string `yaml:"images,omitempty"`
}

func loadContainerImages() []string {
	var containerImageList ImageList

	yamlFile, err := os.ReadFile("containers.yaml")
	if err != nil {
		fmt.Printf("failed to load container image list, %v\n", err)
		os.Exit(1)
	}
	yaml.Unmarshal(yamlFile, &containerImageList)
	return containerImageList.Images
}

func pullImage(wg *sync.WaitGroup, client *client.Client, ctx context.Context, image string) {
	defer wg.Done()

	r, err := client.ImagePull(ctx, image, imagePullOptions)
	if err != nil {
		fmt.Printf("Failed to pull image %s: %v\n", image, err)
		return
	}
	defer r.Close()

	if showPullDetail {
		io.Copy(os.Stdout, r)
	}
}

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("Error creating Docker client: %v\n", err)
		fmt.Println("Please ensure Docker is running and accessible.")
		os.Exit(1)
	}

	// Test the connection to Docker daemon
	_, err = cli.Ping(ctx)
	if err != nil {
		fmt.Printf("Cannot connect to Docker daemon: %v\n", err)
		fmt.Println("Please check if:")
		fmt.Println("  1. Docker Desktop is running")
		fmt.Println("  2. Docker daemon is accessible at unix:///var/run/docker.sock")
		fmt.Println("  3. You have permission to access Docker")
		os.Exit(1)
	}

	var containerImages = loadContainerImages()
	var wg sync.WaitGroup
	for _, containerImage := range containerImages {
		wg.Add(1)
		go pullImage(&wg, cli, ctx, containerImage)
	}
	wg.Wait()

	if cleanupAfterTest {
		for _, v := range containerImages {
			if _, err := cli.ImageRemove(ctx, v, imageRemoveOptions); err != nil {
				if !strings.Contains(err.Error(), "No such image:") {
					fmt.Printf("%+v\n", err)
				}
			}
		}
	}
}
