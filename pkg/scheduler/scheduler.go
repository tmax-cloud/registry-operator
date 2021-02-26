package scheduler

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/pkg/scheduler/pool"
	"github.com/tmax-cloud/registry-operator/pkg/structs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Scheduler runs functions for RegistryJob

const (
	maxConcurrentJob = 1
)

var log = logf.Log.WithName("job-scheduler")

// New is a constructor for a Scheduler
func New(c client.Client, s *runtime.Scheme) *Scheduler {
	log.Info("New scheduler")
	sch := &Scheduler{
		k8sClient: c,
		scheme:    s,
		caller:    make(chan struct{}, 1),
	}
	sch.jobPool = pool.NewJobPool(sch.caller, priorityBasedFifoCompare)
	sch.handlers = Handlers{}
	go sch.start()
	return sch
}

// Handler is an interface that scheduler can call
type Handler interface {
	// Handle is a function to be called by scheduler
	Handle(object types.NamespacedName) error
}

// Handlers is a map of handler
type Handlers map[v1.RegistryJobType]Handler

// Scheduler watches RegistryJobs and calls the job handlers, considering how many runs are running (in a jobPool)
type Scheduler struct {
	k8sClient client.Client
	scheme    *runtime.Scheme

	jobPool  *pool.JobPool
	handlers Handlers

	// Buffered channel with capacity 1
	// Since scheduler lists resources by itself, the actual scheduling logic should be executed only once even when
	// Schedule is called for several times
	caller chan struct{}
}

// Notify notifies scheduler to sync
func (s Scheduler) Notify(job *v1.RegistryJob) {
	s.jobPool.Lock()
	s.jobPool.SyncJob(job)
	s.jobPool.Unlock()
}

func (s Scheduler) start() {
	for range s.caller {
		s.run()
		// Set minimum time gap between scheduling logic
		time.Sleep(3 * time.Second)
	}
}

func (s Scheduler) run() {
	s.jobPool.Lock()
	defer s.jobPool.Unlock()
	log.Info("scheduling...")
	availableCnt := maxConcurrentJob - s.jobPool.Running.Len()

	// If the number of running jobs is greater or equals to the max concurrent job, no scheduling is allowed
	if availableCnt <= 0 {
		log.Info("Max number of jobs are already running")
		return
	}

	// Schedule if available
	s.jobPool.Pending.ForEach(s.schedulePending(&availableCnt))
}

func (s *Scheduler) schedulePending(availableCnt *int) func(structs.Item) {
	return func(item structs.Item) {
		if *availableCnt <= 0 {
			return
		}
		jobNode, ok := item.(*pool.JobNode)
		if !ok {
			return
		}

		log.Info(fmt.Sprintf("Scheduled %s / %s / %s", jobNode.Name, jobNode.Namespace, jobNode.CreationTimestamp))
		go s.executeJob(jobNode.RegistryJob)

		*availableCnt = *availableCnt - 1
	}
}

// RegisterHandler registers a handler which the scheduler can call
func (s *Scheduler) RegisterHandler(newType v1.RegistryJobType, handler Handler) error {
	_, exist := s.handlers[newType]
	if exist {
		err := fmt.Errorf("%s type func is already exist", newType)
		log.Error(err, "failed to register handler")
		return err
	}

	s.handlers[newType] = handler

	return nil
}

func (s *Scheduler) executeJob(job *v1.RegistryJob) {
	jobLog := log.WithValues("job namespace", job.Namespace, "job name", job.Name)
	jobLog.Info("executing job")

	// Set as running
	if err := s.patchJobStarted(job); err != nil {
		jobLog.Error(err, "")
	}

	state := v1.RegistryJobStateCompleted
	msg := ""

	if job.Spec.Claim == nil || job.Spec.Claim.JobType == "" || job.Spec.Claim.HandleObject.Name == "" {
		err := fmt.Errorf("invalid claim")
		jobLog.Error(err, "something in the claim is empty")
		state = v1.RegistryJobStateFailed
		msg = err.Error()
		if err := s.patchJobCompleted(job, state, msg); err != nil {
			jobLog.Error(err, "")
			return
		}
	}

	handler, exist := s.handlers[job.Spec.Claim.JobType]
	if !exist {
		err := fmt.Errorf("%s type func is not exist", job.Spec.Claim.JobType)
		jobLog.Error(err, "failed to get handle function")
		state = v1.RegistryJobStateFailed
		msg = err.Error()
		if err := s.patchJobCompleted(job, state, msg); err != nil {
			jobLog.Error(err, "")
		}
		return
	}

	object := types.NamespacedName{Name: job.Spec.Claim.HandleObject.Name, Namespace: job.Namespace}
	if err := handler.Handle(object); err != nil {
		jobLog.Error(err, "failed to execute job", "job type", job.Spec.Claim.JobType, "object namespace", object.Namespace, "object name", object.Name)
		state = v1.RegistryJobStateFailed
		msg = err.Error()
		if err := s.patchJobCompleted(job, state, msg); err != nil {
			jobLog.Error(err, "")
		}
		return
	}

	// Set as complete
	if err := s.patchJobCompleted(job, state, msg); err != nil {
		jobLog.Error(err, "")
	}
}

func (s *Scheduler) patchJobStarted(job *v1.RegistryJob) error {
	original := job.DeepCopy()

	job.Status.State = v1.RegistryJobStateRunning
	job.Status.StartTime = &metav1.Time{Time: time.Now()}

	p := client.MergeFrom(original)
	return s.k8sClient.Status().Patch(context.Background(), job, p)
}

func (s *Scheduler) patchJobCompleted(job *v1.RegistryJob, state v1.RegistryJobState, message string) error {
	original := job.DeepCopy()

	job.Status.State = state
	job.Status.Message = message
	job.Status.CompletionTime = &metav1.Time{Time: time.Now()}

	p := client.MergeFrom(original)
	return s.k8sClient.Status().Patch(context.Background(), job, p)
}
