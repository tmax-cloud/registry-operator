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
)

func defaultValues() map[string]string {
	values := map[string]string{}

	values[ConfigRegistryImage] = "registry:2.7.1"
	values[ConfigNotaryServerImage] = "tmaxcloudck/notary_server:0.6.2-rc1"
	values[ConfigNotarySignerImage] = "tmaxcloudck/notary_signer:0.6.2-rc1"
	values[ConfigNotaryDBImage] = "tmaxcloudck/notary_mysql:0.6.2-rc1"
	values[ConfigRegistryCPU] = "0.1"
	values[ConfigRegistryMemory] = "512Mi"
	values[ConfigNotaryServerCPU] = "0.1"
	values[ConfigNotaryServerMemory] = "128Mi"
	values[ConfigNotarySignerCPU] = "0.1"
	values[ConfigNotarySignerMemory] = "128Mi"
	values[ConfigNotaryDBCPU] = "0.1"
	values[ConfigNotaryDBMemory] = "256Mi"
	values[ConfigExternalRegistrySyncPeriod] = "*/5 * * * *"

	// If IMAGE_REGISTRY is set, it assumes the necessary images are in the registry.
	registry := Config.GetString(ConfigImageRegistry)
	if registry != "" {
		values[ConfigRegistryImage] = fmt.Sprintf("%s/%s", registry, values[ConfigRegistryImage])
		values[ConfigNotaryServerImage] = fmt.Sprintf("%s/%s", registry, values[ConfigNotaryServerImage])
		values[ConfigNotarySignerImage] = fmt.Sprintf("%s/%s", registry, values[ConfigNotarySignerImage])
		values[ConfigNotaryDBImage] = fmt.Sprintf("%s/%s", registry, values[ConfigNotaryDBImage])
	}

	imagePullSecret := Config.GetString(ConfigImageRegistryPullRequest)
	if imagePullSecret != "" {
		values[ConfigRegistryImagePullSecret] = imagePullSecret
		values[ConfigNotaryServerImagePullSecret] = imagePullSecret
		values[ConfigNotarySignerImagePullSecret] = imagePullSecret
		values[ConfigNotaryDBImagePullSecret] = imagePullSecret
	}

	return values
}

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
func InitEnv() {
	Config.SetDefault("operator.namespace", "registry-system")
	defaults := defaultValues()
	for key, val := range defaults {
		Config.SetDefault(key, val)
	}
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
