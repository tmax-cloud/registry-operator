package image

import (
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
)

func TestSetImage(t *testing.T) {
	image, err := NewImage("reg-test.tmax-registry.registry.172.22.11.10.nip.io/alpine:3", "https://reg-test.tmax-registry.registry.172.22.11.10.nip.io", "", nil)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "reg-test.tmax-registry.registry.172.22.11.10.nip.io", image.Host)
	assert.Equal(t, "alpine", image.Name)
	assert.Equal(t, "3", image.Tag)

	if err := image.SetImage("reg-test.tmax-registry.registry.172.22.11.10.nip.io/alpine:3"); err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "reg-test.tmax-registry.registry.172.22.11.10.nip.io", image.Host)
	assert.Equal(t, "alpine", image.Name)
	assert.Equal(t, "3", image.Tag)

	image, err = NewImage("registry-1.docker.io/library/alpine:3", "", "", nil)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "registry-1.docker.io", image.Host)
	assert.Equal(t, "library/alpine", image.Name)
	assert.Equal(t, "3", image.Tag)

	image, err = NewImage("docker.io/library/alpine:3", "", "", nil)
	if err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "docker.io", image.Host)
	assert.Equal(t, "library/alpine", image.Name)
	assert.Equal(t, "3", image.Tag)

	if err := image.SetImage("library/alpine:3"); err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "docker.io", image.Host)
	assert.Equal(t, "library/alpine", image.Name)
	assert.Equal(t, "3", image.Tag)

	if err := image.SetImage("alpine:3"); err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "docker.io", image.Host)
	assert.Equal(t, "library/alpine", image.Name)
	assert.Equal(t, "3", image.Tag)

	if err := image.SetImage("alpine"); err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "docker.io", image.Host)
	assert.Equal(t, "library/alpine", image.Name)
	assert.Equal(t, "latest", image.Tag)

	if err := image.SetImage("reg-test.tmax-registry.registry.172.22.11.10.nip.io/alpine@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"); err != nil {
		fmt.Println(err.Error())
		t.Fatal()
	}
	assert.Equal(t, "reg-test.tmax-registry.registry.172.22.11.10.nip.io", image.Host)
	assert.Equal(t, "alpine", image.Name)
	assert.Equal(t, "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc", image.Digest)
	assert.Equal(t, "", image.Tag)
}
