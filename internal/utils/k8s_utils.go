package utils

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	authorization "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func CheckAndCreateObject(client client.Client, namespacedName types.NamespacedName, obj runtime.Object) error {
	resourceType := reflect.TypeOf(obj).String()
	reqLogger := log.Log.WithValues(resourceType+".Namespace", namespacedName.Namespace, resourceType+".Name", namespacedName.Name)

	err := client.Get(context.TODO(), namespacedName, obj)
	if err != nil && k8serrors.IsNotFound(err) {
		reqLogger.Info("Creating")
		if err = client.Create(context.TODO(), obj); err != nil {
			reqLogger.Error(err, "Error creating")
			return err
		}
	} else if err != nil {
		reqLogger.Error(err, "Error getting status")
		return err
	} else {
		reqLogger.Info("Already Exist")
	}
	return nil
}

type Patcher struct {
	PatchType types.PatchType
	DataBytes []byte
}

func (p *Patcher) Type() types.PatchType {
	return p.PatchType
}

func (p *Patcher) Data(obj runtime.Object) ([]byte, error) {
	return p.DataBytes, nil
}

func BuildServiceHostname(name, namespace string) string {
	return strings.Join([]string{name, namespace, "svc", "cluster", "local"}, ".")
}

func Client(options client.Options) (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return client.New(cfg, options)
}

func AuthClient() (*authorization.AuthorizationV1Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return authorization.NewForConfig(cfg)
}

func Namespace() (string, error) {
	nsPath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	if FileExists(nsPath) {
		// Running in k8s cluster
		nsBytes, err := ioutil.ReadFile(nsPath)
		if err != nil {
			return "", fmt.Errorf("could not read file %s", nsPath)
		}
		return string(nsBytes), nil
	} else {
		// Not running in k8s cluster (may be running locally)
		ns := os.Getenv("NAMESPACE")
		if ns == "" {
			ns = "default"
		}
		return ns, nil
	}
}

func OperatorServiceName() string {
	svcName := os.Getenv("OPERATOR_SERVICE_NAME")
	if svcName == "" {
		svcName = "image-signer"
	}
	return svcName
}

func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type dockerConfig map[string]dockerConfigEntry

type dockerConfigEntry struct {
	Auth     string `json:"auth"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type dockerConfigJson struct {
	Auths dockerConfig `json:"auths"`
}

func ParseBasicAuth(sec *corev1.Secret, host string) (string, error) {
	if sec == nil {
		return "", fmt.Errorf("cannot get secret")
	}
	if sec.Type != corev1.SecretTypeDockerConfigJson {
		return "", fmt.Errorf("secret is not a docker config type")
	}
	data, ok := sec.Data[corev1.DockerConfigJsonKey]
	if !ok {
		return "", fmt.Errorf("secret is not a docker config type")
	}

	cfg := &dockerConfigJson{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return "", err
	}

	for k, v := range cfg.Auths {
		if k == host {
			if v.Auth != "" {
				return v.Auth, nil
			} else if v.Username != "" && v.Password != "" {
				return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", v.Username, v.Password))), nil
			}
			return "", fmt.Errorf("cannot find docker credential for %s", host)
		}
	}

	return "", fmt.Errorf("cannot find docker credential for %s", host)
}
