package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	cleanupAfterTest   = true
	showPullDetail     = true
	imagePullOptions   = types.ImagePullOptions{}
	imageRemoveOptions = types.ImageRemoveOptions{true, true}

	containerImages = []string{
		"nginx",
		"httpd",
		"alpine",
	}
)

func main() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("%+v\n", err)
		panic(err)
	}

	var wg sync.WaitGroup
	for _, containerImage := range containerImages {
		wg.Add(1)

		targetImage := containerImage
		go func() {
			defer wg.Done()

			// TODO; should check image existance before pull
			r, err := cli.ImagePull(ctx, targetImage, imagePullOptions)
			if err != nil {
				fmt.Printf("%+v\n", err)
				panic(err)
			}
			defer r.Close()

			if showPullDetail {
				io.Copy(os.Stdout, r)
			}
		}()
	}
	wg.Wait()

	if cleanupAfterTest {
		for _, v := range containerImages {
			if _, err := cli.ImageRemove(ctx, v, imageRemoveOptions); err != nil {
				fmt.Printf("%+v\n", err)
			}
		}
	}
}
