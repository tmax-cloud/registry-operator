package auth

import (
	"errors"
	"reflect"

	secret "github.com/tmax-cloud/registry-operator/internal/auth/secret"
	v1 "k8s.io/api/core/v1"
)

type LoginProvider interface {
	getID() string
	getPassword() string
}

type CertProvider interface {
	getCert() string
	getKey() string
}

func NewLoginProvider(dataSource interface{}) (*LoginProvider, error) {

	var ret *LoginProvider

	switch dataSource.(type) {
	case v1.Secret:
		ret = secret.NewLoginAuth(dataSource.(*v1.Secret))
	default:
		return nil, errors.New("Unsupported data source type: " + reflect.TypeOf(dataSource).Name())
	}

	return ret, nil
}

func NewCertProvider(dataSource interface{}) (*CertProvider, error) {

	var ret *CertProvider

	switch dataSource.(type) {
	case v1.Secret:
		ret = secret.NewCertAuth(dataSource.(*v1.Secret))
	default:
		return nil, errors.New("Unsupported data source type: " + reflect.TypeOf(dataSource).Name())
	}

	return ret, nil
}
