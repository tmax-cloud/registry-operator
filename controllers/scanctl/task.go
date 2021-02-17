package scanctl

// ImageScanRequest 1:1 (has many ImageScanRequest.ScanTarget)
type ScanTask struct {
	// id      string
	jobs      []*ScanJob
	OnStart   func(*ScanTask)
	OnSuccess func(*ScanTask)
	OnFail    func(error)
}

func NewScanTask(jobs []*ScanJob, start func(*ScanTask), success func(*ScanTask), fail func(error)) *ScanTask {
	return &ScanTask{
		// id:      id,
		jobs:      jobs,
		OnStart:   start,
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
