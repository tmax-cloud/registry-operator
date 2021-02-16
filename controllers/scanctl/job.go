package scanctl

import (
	"context"
	"regexp"
	"strings"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
)

type ScanJob struct {
	r      *registry.Registry
	c      *clair.Clair
	images []string
}

func NewScanJob(r *registry.Registry, c *clair.Clair, images []string) *ScanJob {
	return &ScanJob{
		r:      r,
		c:      c,
		images: images,
	}
}

func (j *ScanJob) Run() error {

	reports := make(map[string]*clair.VulnerabilityReport, len(j.images))

	repos, err := j.r.Catalog(context.TODO(), "")
	if err != nil {
		return err
	}

	targets := []string{}

	for _, pattern := range j.images {
		if pattern == "*" {
			targets = repos
			break
		}

		for _, repo := range repos {
			isMatched, _ := regexp.MatchString(pattern, repo)
			if isMatched && !isDuplicated(targets, repo) {
				targets = append(targets, repo)
			}
		}
	}

	for _, imageName := range targets {

		imageFullname := strings.Join([]string{j.r.Domain, imageName}, "/")

		image, err := registry.ParseImage(imageFullname)
		if err != nil {
			return err
		}

		ctx := context.TODO()
		report, err := j.c.Vulnerabilities(ctx, j.r, image.Path, image.Reference())
		if err != nil {
			return err
		}

		reports[imageFullname] = &report
	}

	return nil
}

func isDuplicated(items []string, str string) bool {
	for _, item := range items {
		if item == str {
			return true
		}
	}
	return false
}
