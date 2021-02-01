package auth

import (
	"fmt"
	"relfect"
	"k8s.io/api/core/v1"
)

type AuthProvider interface {
	getID() string
	getPassword() string
}


func New(dataSource interface{}) (AuthProvider, error){

	var ret AuthProvider

	switch(dataSource.Type) {
	case v1.Secret:
		ret = NewSecretAuth(dataSource)
	default:
		return nil, error.New("Unsupported data source type: " + reflect.TypeOf(dataSource).Name())
	}

	return ret, nil
}