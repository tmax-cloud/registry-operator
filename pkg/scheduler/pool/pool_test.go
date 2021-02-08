package pool

import (
	"fmt"
	"github.com/bmizerany/assert"
	v1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/pkg/structs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestJobPool_SyncJob(t *testing.T) {
	ch := make(chan struct{}, 1)
	p := NewJobPool(ch, testCompare)

	now := time.Now()
	testJob1 := jobForTest("1", "default", now)
	testJob2 := jobForTest("2", "default", now)
	testJob3 := jobForTest("3", "default", now)
	testJob4 := jobForTest("4", "default", now)
	testJob5 := jobForTest("5", "default", now)
	testJob6 := jobForTest("6", "default", now)
	testJob7 := jobForTest("6", "l2c-system", now)

	p.SyncJob(testJob1)
	p.SyncJob(testJob2)
	p.SyncJob(testJob3)
	p.SyncJob(testJob4)
	p.SyncJob(testJob5)
	p.SyncJob(testJob6)
	p.SyncJob(testJob7)

	// Initial
	assert.Equal(t, 7, p.Pending.Len(), "state transition isn't done properly")
	assert.Equal(t, 0, p.Running.Len(), "state transition isn't done properly")

	// 3 Running
	testJob3.Status.State = v1.RegistryJobStateRunning
	p.SyncJob(testJob3)
	assert.Equal(t, 6, p.Pending.Len(), "state transition isn't done properly")
	assert.Equal(t, 1, p.Running.Len(), "state transition isn't done properly")

	// 3 Completed
	testJob3.Status.State = v1.RegistryJobStateCompleted
	p.SyncJob(testJob3)
	assert.Equal(t, 6, p.Pending.Len(), "state transition isn't done properly")
	assert.Equal(t, 0, p.Running.Len(), "state transition isn't done properly")
}

func testCompare(_a, _b structs.Item) bool {
	if _a == nil || _b == nil {
		return false
	}
	a, aOk := _a.(*JobNode)
	b, bOk := _b.(*JobNode)
	if !aOk || !bOk {
		return false
	}

	return a.CreationTimestamp.Time.Before(b.CreationTimestamp.Time) || fmt.Sprintf("%s_%s", a.Namespace, a.Name) < fmt.Sprintf("%s_%s", b.Namespace, b.Name)
}

func jobForTest(name, namespace string, created time.Time) *v1.RegistryJob {
	return &v1.RegistryJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{Time: created},
		},
		Status: v1.RegistryJobStatus{
			State: v1.RegistryJobStatePending,
		},
	}
}
