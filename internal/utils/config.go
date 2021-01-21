package utils

import (
	"fmt"

	"github.com/tmax-cloud/registry-operator/internal/common/config"
)

const (
	defaultImageRegistry     = "registry:2.7.1"
	defaultImageNotaryServer = "tmaxcloudck/notary_server:0.6.2-rc1"
	defaultImageNotarySigner = "tmaxcloudck/notary_signer:0.6.2-rc1"
	defaultImageNotaryDB     = "tmaxcloudck/notary_mysql:0.6.2-rc1"
)

// InitEnv sets undefined environments.
// If IMAGE_REGISTRY is set, it assumes the necessary images are in the registry.
func InitEnv() {
	config.Config.SetDefault("operator.namespace", "registry-system")

	registry := config.Config.GetString("image.registry")
	if registry != "" {
		config.Config.SetDefault("registry.image", fmt.Sprintf("%s/%s", registry, defaultImageRegistry))
		config.Config.SetDefault("notary.server.image", fmt.Sprintf("%s/%s", registry, defaultImageNotaryServer))
		config.Config.SetDefault("notary.signer.image", fmt.Sprintf("%s/%s", registry, defaultImageNotarySigner))
		config.Config.SetDefault("notary.db.image", fmt.Sprintf("%s/%s", registry, defaultImageNotaryDB))

		imagePullSecret := config.Config.GetString("image.registry_pull_secret")
		config.Config.SetDefault("registry.image_pull_secret", imagePullSecret)
		config.Config.SetDefault("notary.server.image_pull_secret", imagePullSecret)
		config.Config.SetDefault("notary.signer.image_pull_secret", imagePullSecret)
		config.Config.SetDefault("notary.db.image_pull_secret", imagePullSecret)

		return
	}

	config.Config.SetDefault("registry.image", defaultImageRegistry)
	config.Config.SetDefault("notary.server.image", defaultImageNotaryServer)
	config.Config.SetDefault("notary.signer.image", defaultImageNotarySigner)
	config.Config.SetDefault("notary.db.image", defaultImageNotaryDB)
}
