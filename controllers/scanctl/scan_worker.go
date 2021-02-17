package scanctl

import (
	"fmt"
	"sync"
)

type ScanWorker struct {
	workqueue chan *ScanTask
	queueSize int
	nWorkers  int
	wait      *sync.WaitGroup
	stopCh    chan bool
}

func NewScanWorker(queueSize, nWorkers int) *ScanWorker {
	return &ScanWorker{
		workqueue: make(chan *ScanTask, queueSize),
		queueSize: queueSize,
		nWorkers:  nWorkers,
		wait:      &sync.WaitGroup{},
		stopCh:    make(chan bool, 1),
	}
}

func (w *ScanWorker) Submit(o *ScanTask) *ScanWorker {
	w.workqueue <- o
	return w
}

func (w *ScanWorker) Start() {
	for i := 0; i < w.nWorkers; i++ {
		w.wait.Add(1)

		go func() {
			defer w.wait.Done()
			for {
				select {
				case task, isOpened := <-w.workqueue:
					if !isOpened {
						fmt.Printf("** [ScanWorker]: Terminate\n")
						return
					}

					fmt.Printf("** [ScanWorker]: Start Task\n")
					task.OnStart(task)

					var err error
					for _, job := range task.Jobs() {
						if err = job.Run(); err != nil {
							break
						}
					}
					if err != nil {
						fmt.Printf("** [ScanWorker]: Fail Task\n")
						task.OnFail(err)
						break
					}

					fmt.Printf("** [ScanWorker]: Finish Task\n")
					task.OnSuccess(task)
				}
			}
		}()
	}

	go func() {
		select {
		case <-w.stopCh:
			close(w.workqueue)
		}
	}()
}

func (w *ScanWorker) Stop() {
	w.stopCh <- true
}
