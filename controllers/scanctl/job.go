package scanctl

import (
	"context"
	"path"
	"regexp"
	"strings"

	"github.com/genuinetools/reg/clair"
	"github.com/genuinetools/reg/registry"
)

// 1:1 ImageScanRequest.ScanTarget
type ScanJob struct {
	r                 *registry.Registry
	c                 *clair.Clair
	images            []string
	maxAllowedVuls    int
	result            map[string]*clair.VulnerabilityReport
	SendReportEnabled bool
}

func NewScanJob(r *registry.Registry, c *clair.Clair, images []string, nAllowVuls int, sendReport bool) *ScanJob {
	return &ScanJob{
		r:                 r,
		c:                 c,
		images:            images,
		maxAllowedVuls:    nAllowVuls,
		SendReportEnabled: sendReport,
	}
}

func (j *ScanJob) Result() map[string]*clair.VulnerabilityReport {
	return j.result
}

func (j *ScanJob) MaxVuls() int {
	return j.maxAllowedVuls
}

func (j *ScanJob) Run() error {
	isContainsPattern := false
	for _, image := range j.images {
		if strings.ContainsAny("*?", image) {
			isContainsPattern = true
			break
		}
	}
	targets := []string{}
	if isContainsPattern {
		// FIXME: Not possible in the case of docker.io
		repositories, err := j.r.Catalog(context.TODO(), "")
		if err != nil {
			return err
		}
		for _, repo := range repositories {
			for _, image := range j.images {
				if isMatch, _ := regexp.MatchString(convertToRegexp(image), repo); isMatch {
					targets = append(targets, repo)
					break
				}
			}
		}
	} else {
		targets = j.images
	}
	vuls := make(map[string]*clair.VulnerabilityReport, len(j.images))
	for _, target := range targets {
		imagePath := path.Join(j.r.Domain, target)
		img, err := registry.ParseImage(imagePath)
		if err != nil {
			return err
		}
		vul, err := j.c.Vulnerabilities(context.TODO(), j.r, img.Path, img.Reference())
		if err != nil {
			return err
		}
		vuls[imagePath] = &vul
	}
	j.result = vuls
	return nil
}

func convertToRegexp(s string) string {
	c1 := strings.ReplaceAll(s, "?", ".")
	c2 := strings.ReplaceAll(c1, "*", "[[:alnum:]]")
	return c2
}
