package dockerhub

import (
	"errors"
	"strings"

	"github.com/docker/distribution/reference"
)

func ParseName(fullName string) (namespace, repository string, err error) {
	named, e := reference.ParseNormalizedNamed(fullName)
	if e != nil {
		err = e
		return
	}
	name := reference.Path(named)
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		err = errors.New("name must be namespace/repository")
		return
	}
	namespace = parts[0]
	repository = parts[1]
	return
}
