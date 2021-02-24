package scanctl

import (
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
				task, isOpened := <-w.workqueue
				if !isOpened {
					return
				}

				task.OnStart(task)
				var err error
				for _, job := range task.Jobs() {
					if err = job.Run(); err != nil {
						break
					}
				}
				if err != nil {
					task.OnFail(err)
					break
				}
				task.OnSuccess(task)
			}
		}()
	}

	go func() {
		<-w.stopCh
		close(w.workqueue)
	}()
}

func (w *ScanWorker) Stop() {
	w.stopCh <- true
}
