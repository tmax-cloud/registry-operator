package utils

import (
	"fmt"
	"os"
)

const (
	defaultImageRegistry     = "registry:2.7.1"
	defaultImageNotaryServer = "tmaxcloudck/notary_server:0.6.2-rc1"
	defaultImageNotarySigner = "tmaxcloudck/notary_signer:0.6.2-rc1"
	defaultImageNotaryDB     = "tmaxcloudck/notary_mysql:0.6.2-rc1"
)

func setUndefinedEnv(key, value string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, value)
	}
}

// InitEnv sets undefined environments.
// If IMAGE_REGISTRY is set, it assumes the necessary images are in the registry.
func InitEnv() {
	registry := os.Getenv("IMAGE_REGISTRY")
	if registry != "" {
		setUndefinedEnv("REGISTRY_IMAGE", fmt.Sprintf("%s/%s", registry, defaultImageRegistry))
		setUndefinedEnv("NOTARY_SERVER_IMAGE", fmt.Sprintf("%s/%s", registry, defaultImageNotaryServer))
		setUndefinedEnv("NOTARY_SIGNER_IMAGE", fmt.Sprintf("%s/%s", registry, defaultImageNotarySigner))
		setUndefinedEnv("NOTARY_DB_IMAGE", fmt.Sprintf("%s/%s", registry, defaultImageNotaryDB))

		setUndefinedEnv("REGISTRY_IMAGE_PULL_SECRET", os.Getenv("IMAGE_REGISTRY_PULL_SECRET"))
		setUndefinedEnv("NOTARY_SERVER_IMAGE_PULL_SECRET", os.Getenv("IMAGE_REGISTRY_PULL_SECRET"))
		setUndefinedEnv("NOTARY_SIGNER_IMAGE_PULL_SECRET", os.Getenv("IMAGE_REGISTRY_PULL_SECRET"))
		setUndefinedEnv("NOTARY_DB_IMAGE_PULL_SECRET", os.Getenv("IMAGE_REGISTRY_PULL_SECRET"))

		return
	}

	setUndefinedEnv("REGISTRY_IMAGE", defaultImageRegistry)
	setUndefinedEnv("NOTARY_SERVER_IMAGE", defaultImageNotaryServer)
	setUndefinedEnv("NOTARY_SIGNER_IMAGE", defaultImageNotarySigner)
	setUndefinedEnv("NOTARY_DB_IMAGE", defaultImageNotaryDB)
}

func PrintEnv() {
	envs := []string{
		"IMAGE_REGISTRY", "REGISTRY_IMAGE", "NOTARY_SERVER_IMAGE", "NOTARY_SIGNER_IMAGE", "NOTARY_DB_IMAGE",
		"REGISTRY_IMAGE_PULL_SECRET", "NOTARY_SERVER_IMAGE_PULL_SECRET", "NOTARY_SIGNER_IMAGE_PULL_SECRET", "NOTARY_DB_IMAGE_PULL_SECRET",
	}

	fmt.Println("==================== Init Env Start ====================")
	for _, env := range envs {
		fmt.Printf("%-25s: %s\n", env, os.Getenv(env))
	}
	fmt.Println("==================== Init Env End ====================")
}
