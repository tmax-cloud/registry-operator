package v1

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	authorization "k8s.io/api/authorization/v1"

	"github.com/tmax-cloud/registry-operator/internal/utils"
)

const (
	UserHeader   = "X-Remote-User"
	GroupHeader  = "X-Remote-Group"
	ExtrasHeader = "X-Remote-Extra-"
)

func authenticate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
			_ = utils.RespondError(w, http.StatusUnauthorized, "is not https or there is no peer certificate")
			return
		}
		h.ServeHTTP(w, req)
	})
}

func reviewAccess(req *http.Request) error {
	userName, err := getUserName(req.Header)
	if err != nil {
		return err
	}

	userGroups, err := getUserGroup(req.Header)
	if err != nil {
		return err
	}

	userExtras := getUserExtras(req.Header)

	// URL : /apis/registry.tmax.io/v1/ImageSigner/<resource name>/keys
	subPaths := strings.Split(req.URL.Path, "/")
	if len(subPaths) != 7 {
		return fmt.Errorf("URL should be in form of '/apis/registry.tmax.io/v1/ImageSigner/<resource name>/keys'")
	}
	resource := subPaths[4]
	subResource := subPaths[6]

	vars := mux.Vars(req)

	resourceName, nameExist := vars[ResourceParamKey]
	if !nameExist {
		return fmt.Errorf("url is malformed")
	}

	r := &authorization.SubjectAccessReview{
		Spec: authorization.SubjectAccessReviewSpec{
			User:   userName,
			Groups: userGroups,
			Extra:  userExtras,
			ResourceAttributes: &authorization.ResourceAttributes{
				Name:        resourceName,
				Group:       ApiGroup,
				Version:     ApiVersion,
				Resource:    resource,
				Subresource: subResource,
				Verb:        "get",
			},
		},
	}

	result, err := authClient.SubjectAccessReviews().Create(context.TODO(), r, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	if result.Status.Allowed {
		return nil
	}

	return fmt.Errorf(result.Status.Reason)
}

func getUserName(header http.Header) (string, error) {
	for k, v := range header {
		if k == UserHeader {
			return v[0], nil
		}
	}
	return "", fmt.Errorf("no header %s", UserHeader)
}

func getUserGroup(header http.Header) ([]string, error) {
	for k, v := range header {
		if k == UserHeader {
			return v, nil
		}
	}
	return nil, fmt.Errorf("no header %s", GroupHeader)
}

func getUserExtras(header http.Header) map[string]authorization.ExtraValue {
	extras := map[string]authorization.ExtraValue{}

	for k, v := range header {
		if strings.HasPrefix(k, ExtrasHeader) {
			extras[strings.TrimPrefix(k, ExtrasHeader)] = v
		}
	}

	return extras
}
