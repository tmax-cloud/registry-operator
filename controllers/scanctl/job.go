package scanctl

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
)

// 1:1 ImageScanRequest.ScanTarget
type ScanJob struct {
	r              *registry.Registry
	c              *clair.Clair
	images         []string
	maxAllowedVuls int
	result         map[string]*clair.VulnerabilityReport
	SendReport     bool
}

func NewScanJob(r *registry.Registry, c *clair.Clair, images []string, nAllowVuls int, sendReport bool) *ScanJob {
	return &ScanJob{
		r:              r,
		c:              c,
		images:         images,
		maxAllowedVuls: nAllowVuls,
		SendReport:     sendReport,
	}
}

func (j *ScanJob) Result() map[string]*clair.VulnerabilityReport {
	return j.result
}

func (j *ScanJob) MaxVuls() int {
	return j.maxAllowedVuls
}

func (j *ScanJob) Run() error {

	repos, err := j.r.Catalog(context.TODO(), "")
	if err != nil {
		return err
	}

	fmt.Printf("**** [ScanJob]: repogitories: %s\n", repos)
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

	fmt.Printf("**** [ScanJob]: Matching targets: %s\n", targets)
	reports := make(map[string]*clair.VulnerabilityReport, len(j.images))
	for _, imageName := range targets {
		imageFullname := strings.Join([]string{j.r.Domain, imageName}, "/")
		image, err := registry.ParseImage(imageFullname)
		if err != nil {
			return err
		}

		fmt.Printf("**** [ScanJob]: Start scan: %s\n", imageFullname)
		ctx := context.TODO()
		report, err := j.c.Vulnerabilities(ctx, j.r, image.Path, image.Reference())
		if err != nil {
			return err
		}

		fmt.Printf("**** [ScanJob]: Finished scan: %s\n", imageFullname)
		reports[imageFullname] = &report
	}

	fmt.Printf("**** [ScanJob]: Send Result\n")
	j.result = reports
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
