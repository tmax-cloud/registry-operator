package schemes

import (
	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
)

type SubresourceType int

const (
	NotaryServerPrefix = "server-"
	NotarySignerPrefix = "signer-"
	NotaryDBPrefix     = "db-"
)

const (
	SubTypeNotaryDBPod = SubresourceType(iota)
	SubTypeNotaryDBPVC
	SubTypeNotaryDBService
	SubTypeNotaryServerIngress
	SubTypeNotaryServerPod
	SubTypeNotaryServerSecret
	SubTypeNotaryServerService
	SubTypeNotarySignerPod
	SubTypeNotarySignerSecret
	SubTypeNotarySignerService
)

// SubresourceName returns Notary's or Registry's subresource name
func SubresourceName(subresource interface{}, subresourceType SubresourceType) string {
	switch res := subresource.(type) {
	case *regv1.Notary:
		switch subresourceType {
		// Notary DB
		case SubTypeNotaryDBPod, SubTypeNotaryDBPVC, SubTypeNotaryDBService:
			return regv1.K8sPrefix + regv1.K8sNotaryPrefix + NotaryDBPrefix + res.Name

		// Notary Server
		case SubTypeNotaryServerIngress, SubTypeNotaryServerPod, SubTypeNotaryServerSecret, SubTypeNotaryServerService:
			return regv1.K8sPrefix + regv1.K8sNotaryPrefix + NotaryServerPrefix + res.Name

		// Notary signer
		case SubTypeNotarySignerPod, SubTypeNotarySignerSecret, SubTypeNotarySignerService:
			return regv1.K8sPrefix + regv1.K8sNotaryPrefix + NotarySignerPrefix + res.Name
		}

		// TODO: *regv1.Registry
	}

	return ""
}
