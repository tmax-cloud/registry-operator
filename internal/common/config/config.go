package config

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config is registry's environment configuration
var Config *viper.Viper

const (
	configFilePath = "/registry-operator/config/manager_config.yaml"

	defaultImageRegistry     = "registry:2.7.1"
	defaultImageNotaryServer = "tmaxcloudck/notary_server:0.6.2-rc1"
	defaultImageNotarySigner = "tmaxcloudck/notary_signer:0.6.2-rc1"
	defaultImageNotaryDB     = "tmaxcloudck/notary_mysql:0.6.2-rc1"
)

func init() {
	var configFile string
	Config = viper.New()
	Config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	Config.AutomaticEnv()

	configFile = os.Getenv("REGISTRY_CONFIG_FILE")
	if configFile == "" {
		configFile = configFilePath
	}
	filename := path.Base(configFile)
	ext := path.Ext(configFile)
	configPath := path.Dir(configFile)

	Config.SetConfigType(strings.TrimPrefix(ext, "."))
	Config.SetConfigName(strings.TrimSuffix(filename, ext))
	Config.AddConfigPath(configPath)

	if err := Config.ReadInConfig(); err != nil {
		fmt.Println(err.Error())
		return
	}
}

// InitEnv sets undefined environments.
// If IMAGE_REGISTRY is set, it assumes the necessary images are in the registry.
func InitEnv() {
	Config.SetDefault("operator.namespace", "registry-system")

	registry := Config.GetString(ConfigImageRegistry)
	if registry != "" {
		Config.SetDefault(ConfigRegistryImage, fmt.Sprintf("%s/%s", registry, defaultImageRegistry))
		Config.SetDefault(ConfigNotaryServerImage, fmt.Sprintf("%s/%s", registry, defaultImageNotaryServer))
		Config.SetDefault(ConfigNotarySignerImage, fmt.Sprintf("%s/%s", registry, defaultImageNotarySigner))
		Config.SetDefault(ConfigNotaryDBImage, fmt.Sprintf("%s/%s", registry, defaultImageNotaryDB))

		imagePullSecret := Config.GetString(ConfigImageRegistryPullRequest)
		Config.SetDefault(ConfigRegistryImagePullSecret, imagePullSecret)
		Config.SetDefault(ConfigNotaryServerImagePullSecret, imagePullSecret)
		Config.SetDefault(ConfigNotarySignerImagePullSecret, imagePullSecret)
		Config.SetDefault(ConfigNotaryDBImagePullSecret, imagePullSecret)

		return
	}

	Config.SetDefault(ConfigRegistryImage, defaultImageRegistry)
	Config.SetDefault(ConfigNotaryServerImage, defaultImageNotaryServer)
	Config.SetDefault(ConfigNotarySignerImage, defaultImageNotarySigner)
	Config.SetDefault(ConfigNotaryDBImage, defaultImageNotaryDB)
}

// ReadInConfig is read config file
func ReadInConfig() {
	if Config != nil {
		if err := Config.ReadInConfig(); err != nil {
			fmt.Println(err.Error())
			return
		}
	}
}

// PrintConfig prints configs like key=value
func PrintConfig() {
	for _, key := range Config.AllKeys() {
		fmt.Printf("%s=%s\n", key, Config.GetString(key))
	}
}

// OnConfigChange read config file every syncTime seconds if config file is changed
func OnConfigChange(syncTime time.Duration) {
	Config.WatchConfig()
	Config.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("'%s' config file is changed\n", e.Name)
	})
}
