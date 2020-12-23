package controller

import (
	"context"
	"fmt"

	apiv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/registry"
	"github.com/tmax-cloud/registry-operator/pkg/trust"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var log = ctrl.Log.WithName("signing-controller")

// NewSigningController is a controller for image signing.
// if registryName or registryNamespace is empty string, RegCtl is nil
// if requestNamespace is empty string, get operator's namepsace
func NewSigningController(c client.Client, signer *apiv1.ImageSigner, registryName, registryNamespace string) *SigningController {
	return &SigningController{
		client:      c,
		ImageSigner: signer,
		Regctl:      registry.NewRegCtl(c, registryName, registryNamespace),
	}
}

type SigningController struct {
	client      client.Client
	ImageSigner *apiv1.ImageSigner
	Regctl      *registry.RegCtl
}

func (c *SigningController) CreateRootKey(owner *apiv1.ImageSigner, scheme *runtime.Scheme) (*apiv1.TrustKey, error) {
	log.Info("create root key")

	// Create dummy notary repository
	img, err := trust.NewImage("dummy.com/dummy:dummy", "", "", "", nil)
	if err != nil {
		return nil, err
	}

	not, err := trust.New(img, apiv1.TrustPass{}, fmt.Sprintf("/tmp/%s", utils.RandomString(10)), nil)
	if err != nil {
		return nil, err
	}

	defer not.ClearDir()

	if err := not.CreateRootKey(); err != nil {
		return nil, err
	}

	rootKeyId, rootKey, err := not.ReadRootKey()
	if err != nil {
		return nil, err
	}

	rootPhrase, err := not.GetPassphrase(rootKeyId)
	if err != nil {
		return nil, err
	}

	key := &apiv1.TrustKey{
		ID:         rootKeyId,
		Key:        string(rootKey),
		PassPhrase: rootPhrase,
	}

	if err := c.createRootKey(owner, scheme, key); err != nil {
		log.Error(err, "")
		return nil, err
	}

	log.Info("create root key success")
	return key, nil
}

func (c *SigningController) createRootKey(owner *apiv1.ImageSigner, scheme *runtime.Scheme, trustKey *apiv1.TrustKey) error {
	key := schemes.SignerKey(c.ImageSigner)
	if err := controllerutil.SetOwnerReference(owner, key, scheme); err != nil {
		return err
	}

	key.Spec = apiv1.SignerKeySpec{
		Root: *trustKey,
	}

	if err := c.client.Create(context.TODO(), key); err != nil {
		return err
	}

	return nil
}

func (c *SigningController) SignImage(signerKey *apiv1.SignerKey, img *trust.Image, ca []byte) error {
	// Initialize notary
	passPhrase := signerKey.GetPassPhrase()
	not, err := trust.New(img, passPhrase, fmt.Sprintf("/tmp/%s", utils.RandomString(10)), ca)
	if err != nil {
		return err
	}

	defer not.ClearDir()

	// get trust key &  write it as a file
	log.Info("get trust key")
	rootKey := signerKey.Spec.Root
	if err := not.WriteKey(rootKey.ID, []byte(rootKey.Key)); err != nil {
		return err
	}

	addTargetKey := false
	targetKey, err := signerKey.GetTargetKey(img.GetImageNameWithHost())
	if err != nil {
		addTargetKey = true
	} else {
		if err := not.WriteKey(targetKey.ID, []byte(rootKey.Key)); err != nil {
			return err
		}
	}

	// Sign image
	if err := not.SignImage(); err != nil {
		return err
	}

	newTargetKeyId, newTargetKey, err := not.ReadTargetKey()
	if err != nil {
		return err
	}

	if addTargetKey {
		newPass, err := passPhrase.GetKeyPass(newTargetKeyId)
		if err != nil {
			return err
		}
		newTarget := apiv1.TrustKey{
			ID:         newTargetKeyId,
			Key:        string(newTargetKey),
			PassPhrase: newPass,
		}
		if err := c.addTargetKey(signerKey, img.GetImageNameWithHost(), newTarget); err != nil {
			return err
		}
	}

	return nil
}

func (c *SigningController) addTargetKey(signerKey *apiv1.SignerKey, targetName string, targetKey apiv1.TrustKey) error {
	key2 := signerKey.DeepCopy()
	key2.Spec.Targets[targetName] = targetKey

	if err := c.client.Patch(context.TODO(), signerKey, client.MergeFrom(signerKey)); err != nil {
		return err
	}

	return nil
}
