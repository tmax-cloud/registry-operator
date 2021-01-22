package config

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/viper"
)

// Config is registry's environment configuration
var Config *viper.Viper

const configFilePath = "/registry-operator/config/manager_config.yaml"

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
