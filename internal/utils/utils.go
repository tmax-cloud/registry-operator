package utils

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/operator-framework/operator-lib/status"
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("utils-log")

// Use for GetRegistryLogger
func funcName() string {
	pc := make([]uintptr, 15)
	n := runtime.Callers(4, pc) //Skip: 3 (Callers, getFuncName, GetRegistryLogger, get)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()

	return frame.Function
}

// [TODO] API is not worked well
func GetRegistryLogger(subresource interface{}, resNamespace, resName string) logr.Logger {
	typeName := reflect.TypeOf(subresource).Name()
	funcName := funcName()
	path := strings.Split(funcName, ".")
	funcName = path[len(path)-1]

	return log.Log.WithValues(typeName+".Namespace", resNamespace, typeName+".Name", resName, typeName+".Api", funcName)
}

// SetCondition sets a condition of patch status
func SetCondition(error error, patch interface{}, condition *status.Condition) {
	if error != nil {
		condition.Message = error.Error()
	}
	switch p := patch.(type) {
	case *regv1.Registry:
		p.Status.Conditions.SetCondition(*condition)
	case *regv1.Notary:
		p.Status.Conditions.SetCondition(*condition)
	case *regv1.ExternalRegistry:
		p.Status.Conditions.SetCondition(*condition)
	}
}

type RegistryLogger struct {
	subresource           interface{}
	resNamespace, resName string
}

func NewRegistryLogger(subresource interface{}, resNamespace, resName string) *RegistryLogger {
	logger := &RegistryLogger{}
	logger.subresource = subresource
	logger.resNamespace = resNamespace
	logger.resName = resName

	return logger
}

func (r *RegistryLogger) Info(msg string, keysAndValues ...interface{}) {
	log := GetRegistryLogger(r.subresource, r.resNamespace, r.resName)
	if len(keysAndValues) > 0 {
		log.Info(msg, keysAndValues...)
	} else {
		log.Info(msg)
	}
}

func (r *RegistryLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	log := GetRegistryLogger(r.subresource, r.resNamespace, r.resName)
	if len(keysAndValues) > 0 {
		log.Error(err, msg, keysAndValues...)
	} else {
		log.Error(err, msg)
	}
}

type Diff struct {
	Type  DiffType
	Key   string
	Value interface{}
}

type DiffType string

const (
	Add     DiffType = "Add"
	Replace DiffType = "Replace"
	Remove  DiffType = "Remove"
)

func DiffKeyList(diffList []Diff) []string {
	keyList := []string{}

	for _, d := range diffList {
		keyList = append(keyList, d.Key)
	}

	return keyList
}

func ParseImageName(imageName string) string {
	return strings.ReplaceAll(strings.ReplaceAll(imageName, "/", "-s-"), "_", "-u-")
}

func Contains(arr []string, str string) bool {
	if arr == nil {
		return false
	}

	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
