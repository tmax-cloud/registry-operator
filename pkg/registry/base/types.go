package base

import (
	cmhttp "github.com/tmax-cloud/registry-operator/internal/common/http"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Factory
type Factory struct {
	K8sClient      client.Client
	NamespacedName types.NamespacedName
	Scheme         *runtime.Scheme
	HttpClient     *cmhttp.HttpClient
}
