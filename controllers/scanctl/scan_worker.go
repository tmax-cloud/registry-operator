package scanctl

type ScanWorker struct {
	workqueue chan *ScanTask
	nWorkers  int
	stopCh    chan bool
}

func NewScanWorker(queueSize, nWorkers int) *ScanWorker {
	return &ScanWorker{
		workqueue: make(chan *ScanTask, queueSize),
		nWorkers:  nWorkers,
		stopCh:    make(chan bool, 1),
	}
}

func (w *ScanWorker) Submit(o *ScanTask) *ScanWorker {
	w.workqueue <- o
	return w
}

func (w *ScanWorker) Start() {
	for i := 0; i < w.nWorkers; i++ {
		go func() {
			for {
				task, isOpened := <-w.workqueue
				if !isOpened {
					return
				}
				var err error
				for _, job := range task.Jobs() {
					if err = job.Run(); err != nil {
						break
					}
				}
				if err != nil {
					task.OnFail(err)
				} else {
					task.OnSuccess(task)
				}
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
