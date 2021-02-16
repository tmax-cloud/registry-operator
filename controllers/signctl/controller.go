package signctl

import (
	"context"
	"encoding/base64"
	"fmt"

	apiv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/schemes"
	"github.com/tmax-cloud/registry-operator/internal/utils"
	"github.com/tmax-cloud/registry-operator/pkg/trust"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var log = ctrl.Log.WithName("signing-controller")

// NewSigningController is a controller for image signing.
// if registryName or registryNamespace is empty string, RegCtl is nil
// if requestNamespace is empty string, get operator's namepsace
func NewSigningController(c client.Client, scheme *runtime.Scheme, signer *apiv1.ImageSigner, registryName, registryNamespace string) *SigningController {
	return &SigningController{
		client:      c,
		ImageSigner: signer,
		Regctl:      NewRegCtl(c, registryName, registryNamespace),
		Scheme:      scheme,
	}
}

type SigningController struct {
	client      client.Client
	ImageSigner *apiv1.ImageSigner
	Regctl      *RegCtl
	Scheme      *runtime.Scheme
}

func (c *SigningController) CreateRootKey(owner *apiv1.ImageSigner, scheme *runtime.Scheme) (*apiv1.TrustKey, error) {
	log.Info("create root key")

	not, err := trust.NewDummy(fmt.Sprintf("/tmp/%s", utils.RandomString(10)))
	if err != nil {
		log.Error(err, "")
		return nil, err
	}

	defer func() {
		if err := not.ClearDir(); err != nil {
			log.Error(err, "")
		}
	}()

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
		Key:        base64.StdEncoding.EncodeToString(rootKey),
		PassPhrase: base64.StdEncoding.EncodeToString([]byte(rootPhrase)),
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
	// Target key
	addTargetKey := false
	targetKey, err := signerKey.GetTargetKey(img.GetImageNameWithHost())
	if err != nil {
		addTargetKey = true
	}

	// Initialize notary
	passPhrase := signerKey.GetPassPhrase()
	not, err := trust.New(img, passPhrase, fmt.Sprintf("/tmp/notary/%s", utils.RandomString(10)), ca, signerKey.Spec.Root, targetKey)
	if err != nil {
		log.Error(err, "")
		return err
	}

	defer func() {
		if err := not.ClearDir(); err != nil {
			log.Error(err, "")
		}
	}()

	// Sign image
	if err := not.SignImage(); err != nil {
		log.Error(err, "")
		return err
	}

	newTargetKeyId, newTargetKey, err := not.ReadTargetKey()
	if err != nil {
		log.Error(err, "")
		return err
	}

	if addTargetKey {
		newPass, err := passPhrase.GetKeyPass(newTargetKeyId)
		if err != nil {
			log.Error(err, "")
			return err
		}
		newTarget := apiv1.TrustKey{
			ID:         newTargetKeyId,
			Key:        base64.StdEncoding.EncodeToString(newTargetKey),
			PassPhrase: base64.StdEncoding.EncodeToString([]byte(newPass)),
		}
		if err := c.addTargetKey(signerKey, img.GetImageNameWithHost(), newTarget); err != nil {
			log.Error(err, "")
			return err
		}
	}

	return nil
}

func (c *SigningController) addTargetKey(signerKey *apiv1.SignerKey, targetName string, targetKey apiv1.TrustKey) error {
	key2 := signerKey.DeepCopy()
	if key2.Spec.Targets == nil {
		key2.Spec.Targets = map[string]apiv1.TrustKey{}
	}
	key2.Spec.Targets[targetName] = targetKey

	if err := c.client.Patch(context.TODO(), key2, client.MergeFrom(signerKey)); err != nil {
		return err
	}

	return nil
}

func (c *SigningController) CreateOwnerRole() error {
	role := &v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.ownerRoleName(),
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups:     []string{"tmax.io"},
				Resources:     []string{"imagesigners"},
				ResourceNames: []string{c.ImageSigner.Name},
				Verbs:         []string{"create", "delete", "get", "list", "patch", "update", "watch"},
			},
		},
	}
	if err := controllerutil.SetControllerReference(c.ImageSigner, role, c.Scheme); err != nil {
		log.Error(err, "SetOwnerReference Failed")
		return err
	}

	if err := c.client.Create(context.TODO(), role); err != nil {
		log.Error(err, "failed to create clusterrole")
		return err
	}

	return nil
}

func (c *SigningController) CreateSignerKeyRole() error {
	role := &v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.signerKeyRoleName(),
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups:     []string{"tmax.io"},
				Resources:     []string{"signerkeys"},
				ResourceNames: []string{c.ImageSigner.Name},
				Verbs:         []string{"get"},
			},
		},
	}
	if err := controllerutil.SetControllerReference(c.ImageSigner, role, c.Scheme); err != nil {
		log.Error(err, "SetOwnerReference Failed")
		return err
	}

	if err := c.client.Create(context.TODO(), role); err != nil {
		log.Error(err, "failed to create clusterrole")
		return err
	}

	return nil
}

func (c *SigningController) CreateOwnerRoleBinding() error {
	labels := map[string]string{}
	labels["object"] = "imagesigner"
	labels["signer"] = c.ImageSigner.Name

	crb := &v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   c.ownerRoleBindingName(),
			Labels: labels,
		},
		Subjects: []v1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     c.ImageSigner.Spec.Owner,
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     c.ownerRoleName(),
		},
	}

	if err := controllerutil.SetControllerReference(c.ImageSigner, crb, c.Scheme); err != nil {
		log.Error(err, "SetOwnerReference Failed")
		return err
	}

	if err := c.client.Create(context.TODO(), crb); err != nil {
		log.Error(err, "failed to create clusterrolebinding")
		return err
	}

	return nil
}

func (c *SigningController) CreateSignerKeyRoleBinding() error {
	labels := map[string]string{}
	labels["object"] = "imagesigner"
	labels["signer"] = c.ImageSigner.Name

	crb := &v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   c.signerKeyRoleBindingName(),
			Labels: labels,
		},
		Subjects: []v1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     c.ImageSigner.Spec.Owner,
			},
		},
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     c.signerKeyRoleName(),
		},
	}

	if err := controllerutil.SetControllerReference(c.ImageSigner, crb, c.Scheme); err != nil {
		log.Error(err, "SetOwnerReference Failed")
		return err
	}

	if err := c.client.Create(context.TODO(), crb); err != nil {
		log.Error(err, "failed to create clusterrolebinding")
		return err
	}

	return nil
}

func (c *SigningController) IsExistOwnerRole() bool {
	req := types.NamespacedName{Name: c.ownerRoleName()}
	role := &v1.ClusterRole{}
	if err := c.client.Get(context.TODO(), req, role); err != nil {
		if errors.IsNotFound(err) {
			return false
		}

		log.Error(err, "failed to get clusterrole")
		return false
	}

	return true
}

func (c *SigningController) IsExistSignerKeyRole() bool {
	req := types.NamespacedName{Name: c.signerKeyRoleName()}
	role := &v1.ClusterRole{}
	if err := c.client.Get(context.TODO(), req, role); err != nil {
		if errors.IsNotFound(err) {
			return false
		}

		log.Error(err, "failed to get clusterrole")
		return false
	}

	return true
}

func (c *SigningController) IsExistOwnerRoleBinding() bool {
	req := types.NamespacedName{Name: c.ownerRoleBindingName()}
	rb := &v1.ClusterRoleBinding{}
	if err := c.client.Get(context.TODO(), req, rb); err != nil {
		if errors.IsNotFound(err) {
			return false
		}

		log.Error(err, "failed to get clusterrolebinding")
		return false
	}

	return true
}

func (c *SigningController) IsExistSignerKeyRoleBinding() bool {
	req := types.NamespacedName{Name: c.signerKeyRoleBindingName()}
	rb := &v1.ClusterRoleBinding{}
	if err := c.client.Get(context.TODO(), req, rb); err != nil {
		if errors.IsNotFound(err) {
			return false
		}

		log.Error(err, "failed to get clusterrolebinding")
		return false
	}

	return true
}

func (c *SigningController) ownerRoleName() string {
	return c.ImageSigner.Name + "-image-signer-owner-role"
}

func (c *SigningController) signerKeyRoleName() string {
	return c.ImageSigner.Name + "-signer-key-role"
}

func (c *SigningController) ownerRoleBindingName() string {
	return c.ImageSigner.Name + "-image-signer-owner-rolebinding"
}

func (c *SigningController) signerKeyRoleBindingName() string {
	return c.ImageSigner.Name + "-signer-key-rolebinding"
}
