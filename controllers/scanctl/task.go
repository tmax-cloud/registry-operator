package scanctl

// ImageScanRequest 1:1 (has many ImageScanRequest.ScanTarget)
type ScanTask struct {
	// id      string
	jobs      []*ScanJob
	OnSuccess func(*ScanTask)
	OnFail    func(error)
}

func NewScanTask(jobs []*ScanJob, success func(*ScanTask), fail func(error)) *ScanTask {
	return &ScanTask{
		// id:      id,
		jobs:      jobs,
		OnSuccess: success,
		OnFail:    fail,
	}
}

// func (t *ScanTask) Name() string {
// 	return t.id
// }

func (t *ScanTask) Jobs() []*ScanJob {
	return t.jobs
}
