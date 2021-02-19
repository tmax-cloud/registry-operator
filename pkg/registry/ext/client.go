package ext

import (
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Logger = log.Log.WithName("ext-registry")
