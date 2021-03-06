package schemes

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/common/config"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type SubresourceType int

const (
	NotaryServerPrefix     = "server-"
	NotarySignerPrefix     = "signer-"
	NotaryDBPrefix         = "db-"
	ExternalRegistryPrefix = "ext-"
	LoginSecretPrefix      = "login-"
	ImageReplicatePrefix   = "repl-"
	SynchronizePrefix      = "sync-"
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

	SubTypeRegistryNotary
	SubTypeRegistryService
	SubTypeRegistryPVC
	SubTypeRegistryDCJSecret
	SubTypeRegistryOpaqueSecret
	SubTypeRegistryTLSSecret
	SubTypeRegistryDeployment
	SubTypeRegistryConfigmap
	SubTypeRegistryIngress

	SubTypeExternalRegistryLoginSecret
	SubTypeExternalRegistryCronJob
	SubTypeExternalRegistryJob

	SubTypeImageReplicateJob
	SubTypeImageReplicateSyncJob
	SubTypeImageReplicateImageSignRequest
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

	case *regv1.Registry:
		switch subresourceType {
		case SubTypeRegistryNotary:
			return res.Name

		case SubTypeRegistryService, SubTypeRegistryPVC, SubTypeRegistryDeployment, SubTypeRegistryOpaqueSecret, SubTypeRegistryConfigmap, SubTypeRegistryIngress:
			return regv1.K8sPrefix + res.Name

		case SubTypeRegistryTLSSecret:
			return regv1.K8sPrefix + regv1.TLSPrefix + res.Name

		case SubTypeRegistryDCJSecret:
			return regv1.K8sPrefix + regv1.K8sRegistryPrefix + res.Name
		}

	case *regv1.ExternalRegistry:
		switch subresourceType {
		case SubTypeExternalRegistryLoginSecret:
			return regv1.K8sPrefix + ExternalRegistryPrefix + LoginSecretPrefix + res.Name
		case SubTypeExternalRegistryCronJob:
			return regv1.K8sPrefix + ExternalRegistryPrefix + res.Name
		case SubTypeExternalRegistryJob:
			return regv1.K8sPrefix + ExternalRegistryPrefix + res.Name + "-" + utils.RandomString(10)
		}

	case *regv1.ImageReplicate:
		switch subresourceType {
		case SubTypeImageReplicateJob:
			return regv1.K8sPrefix + ImageReplicatePrefix + res.Name
		case SubTypeImageReplicateSyncJob:
			return regv1.K8sPrefix + ImageReplicatePrefix + SynchronizePrefix + res.Name
		case SubTypeImageReplicateImageSignRequest:
			if res.Status.ImageSignRequestName != "" {
				return res.Status.ImageSignRequestName
			}
			return regv1.K8sPrefix + ImageReplicatePrefix + res.Name + "-" + utils.RandomString(10)
		}
	}

	return ""
}

const (
	RootCACert = "ca.crt"
	RootCAPriv = "ca.key"
)

func getRootCACertificate(c client.Client) (*x509.Certificate, *rsa.PrivateKey) {
	logger := logf.Log.WithName("cert-util")

	opNamespace := config.Config.GetString("operator.namespace")
	if opNamespace == "" {
		opNamespace = regv1.OperatorNamespace
	}

	rootSecret := corev1.Secret{}
	req := types.NamespacedName{Name: regv1.RegistryRootCASecretName, Namespace: opNamespace}
	if err := c.Get(context.TODO(), req, &rootSecret); err != nil {
		logger.Error(err, "Get Root Secret Error")
		return nil, nil
	}

	block, rest := pem.Decode(rootSecret.Data[RootCACert])
	if len(rest) != 0 {
		logger.Info("Cert is not PEM format", "Rest", rest)
		return nil, nil
	}

	if block == nil {
		logger.Error(errors.New("failed to decode pem data"), "maybe there is no pem data")
		return nil, nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		logger.Error(err, "Parse Root CA block Error")
		return nil, nil
	}

	privBlock, privRest := pem.Decode(rootSecret.Data[RootCAPriv])
	if len(privRest) != 0 {
		logger.Info("Private key is not PEM format", "Rest", privRest)
		return nil, nil
	}

	var key interface{}
	var privKeyErr error

	key, privKeyErr = x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	if privKeyErr != nil {
		key, privKeyErr = x509.ParsePKCS1PrivateKey(privBlock.Bytes)
		if privKeyErr != nil {
			logger.Error(privKeyErr, "Parse private key Error")
			return nil, nil
		}
	}

	return cert, key.(*rsa.PrivateKey)
}
