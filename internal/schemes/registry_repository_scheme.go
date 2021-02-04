package schemes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	regv1 "github.com/tmax-cloud/registry-operator/api/v1"
	"github.com/tmax-cloud/registry-operator/internal/utils"
)

func Repository(reg *regv1.Registry, imageName string, tags []string) *regv1.Repository {
	label := map[string]string{}
	label["app"] = "registry"
	label["apps"] = SubresourceName(reg, SubTypeRegistryDeployment)
	label["registry"] = reg.Name

	versions := []regv1.ImageVersion{}
	for _, ver := range tags {
		newVersion := regv1.ImageVersion{CreatedAt: metav1.Now(), Version: ver, Delete: false}
		versions = append(versions, newVersion)
	}

	return &regv1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepositoryName(imageName, reg.Name),
			Namespace: reg.Namespace,
			Labels:    label,
		},
		Spec: regv1.RepositorySpec{
			Name:     imageName,
			Registry: reg.Name,
			Versions: versions,
		},
	}
}

func RepositoryName(imageName, registryName string) string {
	return utils.ParseImageName(imageName) + "." + registryName
}
