package testutil

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"io/ioutil"
	"testing"
)

func FetchedImage(t *testing.T, image string) string {
	// Create a Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	// Try to fetch the image
	pullResult, err := cli.ImagePull(context.Background(), image, types.ImagePullOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(ioutil.Discard, pullResult)
	if err != nil {
		t.Fatal(err)
	}

	return image
}
