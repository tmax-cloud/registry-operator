package scheduler

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/http"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry/ext/factory"
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
	go sch.start()
	return sch
}

// Scheduler watches RegistryJobs and calls the job handlers, considering how many runs are running (in a jobPool)
type Scheduler struct {
	k8sClient client.Client
	scheme    *runtime.Scheme

	jobPool *pool.JobPool

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

func (s *Scheduler) executeJob(job *v1.RegistryJob) {
	log.Info(fmt.Sprintf("Executing job %s / %s", job.Name, job.Namespace))

	// Set as running
	if err := s.patchJobStarted(job); err != nil {
		log.Error(err, "")
	}

	state := v1.RegistryJobStateCompleted
	msg := ""

	// Sync jobs
	if job.Spec.SyncRepository != nil && job.Spec.SyncRepository.ExternalRegistry.Name != "" {
		// get external registry
		exreg := &v1.ExternalRegistry{}
		exregNamespacedName := types.NamespacedName{Name: job.Spec.SyncRepository.ExternalRegistry.Name, Namespace: job.Namespace}
		if err := s.k8sClient.Get(context.TODO(), exregNamespacedName, exreg); err != nil {
			log.Error(err, "")
		}

		username, password := "", ""
		if exreg.Spec.ImagePullSecret != "" {
			basic, err := utils.GetBasicAuth(exreg.Spec.ImagePullSecret, exreg.Namespace, exreg.Spec.RegistryURL)
			if err != nil {
				log.Error(err, "failed to get basic auth")
			}

			username, password = utils.DecodeBasicAuth(basic)
		}

		var ca []byte
		if exreg.Spec.CertificateSecret != "" {
			data, err := utils.GetCAData(exreg.Spec.CertificateSecret, exreg.Namespace)
			if err != nil {
				log.Error(err, "failed to get ca data")
			}
			ca = data
		}

		syncFactory := factory.NewSyncRegistryFactory(
			s.k8sClient,
			exregNamespacedName,
			s.scheme,
			http.NewHTTPClient(
				exreg.Spec.RegistryURL,
				username, password,
				ca,
				exreg.Spec.Insecure,
			),
		)

		syncClient := syncFactory.Create(exreg.Spec.RegistryType)
		if err := syncClient.Synchronize(); err != nil {
			log.Error(err, "failed to synchronize external registry")
			state = v1.RegistryJobStateFailed
			msg = err.Error()
		}

		log.Info("==========================================")
		log.Info(job.Name)
		log.Info(job.Spec.SyncRepository.ExternalRegistry.Name)
		log.Info("==========================================")
		time.Sleep(10 * time.Second)
	}

	// Set as complete
	if err := s.patchJobCompleted(job, state, msg); err != nil {
		log.Error(err, "")
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
