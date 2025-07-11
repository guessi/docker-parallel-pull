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
	r, err := client.ImagePull(ctx, image, imagePullOptions)
	if err != nil {
		fmt.Printf("%+v\n", err)
		panic(err)
	}
	defer r.Close()

	if showPullDetail {
		io.Copy(os.Stdout, r)
	}
	wg.Done()
}

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("%+v\n", err)
		panic(err)
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
