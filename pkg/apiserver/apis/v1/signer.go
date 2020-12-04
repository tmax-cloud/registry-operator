package v1

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	tmaxiov1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/internal/wrapper"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
)

const (
	SignerApiKeys = "keys"
)

func AddSignerApis(parent *wrapper.RouterWrapper) error {
	signerWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}", SignerKind, ResourceParamKey), nil, nil)
	if err := parent.Add(signerWrapper); err != nil {
		return err
	}

	signerWrapper.Router.Use(Authorize)

	if err := addSignerKeysApi(signerWrapper); err != nil {
		return err
	}
	return nil
}

func addSignerKeysApi(parent *wrapper.RouterWrapper) error {
	keysWrapper := wrapper.New(fmt.Sprintf("/%s", SignerApiKeys), []string{"GET"}, signerKeysHandler)
	if err := parent.Add(keysWrapper); err != nil {
		return err
	}

	return nil
}

func signerKeysHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	resourceName, nameExist := vars[ResourceParamKey]
	if !nameExist {
		_ = utils.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}

	key := &tmaxiov1.SignerKey{}
	if err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: resourceName}, key); err != nil {
		log.Error(err, "cannot get key file")
		if errors.IsNotFound(err) {
			_ = utils.RespondError(w, http.StatusNotFound, fmt.Sprintf("there is no SignerKey %s", resourceName))
		} else {
			_ = utils.RespondError(w, http.StatusInternalServerError, "cannot get SignerKey")
		}
		return
	}

	_ = utils.RespondJSON(w, key)
}
