package pool

import (
	"fmt"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/pkg/structs"
	"sync"
)

// JobPool stores current status of v1.RegistryJobs, who are in Pending status or Running status
// All operations for this pool should be done in thread-safe manner, using Lock and Unlock methods
type JobPool struct {
	jobMap jobMap

	Pending *structs.SortedUniqueList
	Running *structs.SortedUniqueList

	scheduleChan chan struct{}
	lock         sync.Mutex
}

// NewJobPool is a constructor for a JobPool
func NewJobPool(ch chan struct{}, compareFunc structs.CompareFunc) *JobPool {
	return &JobPool{
		jobMap:       jobMap{},
		Pending:      structs.NewSortedUniqueQueue(compareFunc),
		Running:      structs.NewSortedUniqueQueue(nil),
		scheduleChan: ch,
		lock:         sync.Mutex{},
	}
}

// Lock locks JobPool
func (j *JobPool) Lock() {
	j.lock.Lock()
}

// Unlock unlocks JobPool
func (j *JobPool) Unlock() {
	j.lock.Unlock()
}

// SyncJob syncs JobPool with an incoming IntegrationJob job, considering its status
func (j *JobPool) SyncJob(job *v1.RegistryJob) {
	// If job state is not set, return
	if job.Status.State == "" {
		return
	}

	nodeID := getNodeID(job)

	oldStatus := v1.RegistryJobState("")
	newStatus := job.Status.State

	// Make / fetch node pointer
	var node *JobNode
	candidate, exist := j.jobMap[nodeID]
	if exist {
		node = candidate
		oldStatus = candidate.Status.State
		candidate.RegistryJob = job.DeepCopy()
	} else {
		node = &JobNode{
			RegistryJob: job.DeepCopy(),
		}
	}
	j.jobMap[nodeID] = node

	// If there's deletion timestamp, dismiss it
	if node.DeletionTimestamp != nil {
		j.Pending.Delete(node)
		j.Running.Delete(node)
		delete(j.jobMap, nodeID)
		j.sendSchedule()
		return
	}

	// If status is not changed, do nothing
	if exist && oldStatus == newStatus {
		return
	}

	// If it is newly created, put it in proper list
	if !exist {
		switch newStatus {
		case v1.RegistryJobStatePending:
			j.Pending.Add(node)
		case v1.RegistryJobStateRunning:
			j.Running.Add(node)
		}
		j.sendSchedule()
		return
	}

	// Pending -> Running / Failed
	if oldStatus == v1.RegistryJobStatePending {
		j.Pending.Delete(node)
		if newStatus == v1.RegistryJobStateRunning {
			j.Running.Add(node)
		}
		return
	}

	// Running -> The others
	// If it WAS running and not now, dismiss it (it is completed for some reason)
	if oldStatus == v1.RegistryJobStateRunning {
		j.Running.Delete(node)
		if newStatus == v1.RegistryJobStatePending {
			j.Pending.Add(node)
		} else {
			delete(j.jobMap, nodeID)
		}
		j.sendSchedule()
		return
	}
}

func (j *JobPool) sendSchedule() {
	if len(j.scheduleChan) < cap(j.scheduleChan) {
		j.scheduleChan <- struct{}{}
	}
}

// JobNode is a node to be stored in jobMap and JobPool
type JobNode struct {
	*v1.RegistryJob
}

// Equals implements Item's method
func (f *JobNode) Equals(another structs.Item) bool {
	fj, ok := another.(*JobNode)
	if !ok {
		return false
	}
	if f == nil || fj == nil {
		return false
	}
	return f.Name == fj.Name && f.Namespace == fj.Namespace
}

// DeepCopy implements Item's method
func (f *JobNode) DeepCopy() structs.Item {
	return &JobNode{
		RegistryJob: f.RegistryJob.DeepCopy(),
	}
}

// Priority returns Item's priority
func (f *JobNode) Priority() int {
	return f.RegistryJob.Spec.Priority
}

func getNodeID(j *v1.RegistryJob) string {
	return fmt.Sprintf("%s_%s", j.Namespace, j.Name)
}

type jobMap map[string]*JobNode
