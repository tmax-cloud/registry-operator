package scanctl

import (
	"fmt"
	"sync"
)

type ScanWorker struct {
	workqueue  chan []*ScanJob
	queueSize  int
	nWorkers   int
	wait       *sync.WaitGroup
	stopCh     chan bool
	onComplete func(interface{})
}

func NewScanWorker(queueSize, nWorkers int) *ScanWorker {
	return &ScanWorker{
		workqueue:  make(chan []*ScanJob, queueSize),
		queueSize:  queueSize,
		nWorkers:   nWorkers,
		wait:       &sync.WaitGroup{},
		stopCh:     make(chan bool, 1),
		onComplete: func(interface{}) { fmt.Println("**** Job Done ****") }, // Test
	}
}

func (s *ScanWorker) Submit(o []*ScanJob) *ScanWorker {
	s.workqueue <- o
	return s
}

func (s *ScanWorker) Start() {
	for i := 0; i < s.nWorkers; i++ {
		s.wait.Add(1)

		go func() {
			defer s.wait.Done()
			for {
				select {
				case jobs, isOpened := <-s.workqueue:
					if !isOpened {
						fmt.Println("terminate")
						return
					}
					if err := s.doScan(jobs); err != nil {
						fmt.Printf("*** Scan(%s) failed")
					}
					s.onComplete(jobs)
				}
			}
		}()
	}

	go func() {
		select {
		case <-s.stopCh:
			close(s.workqueue)
		}
	}()
}

func (s *ScanWorker) Stop() {
	s.stopCh <- true
}

func (s *ScanWorker) doScan(jobs []*ScanJob) error {
	for _, job := range jobs {
		if err := job.Run(); err != nil {
			return err
		}
	}
	return nil
}
