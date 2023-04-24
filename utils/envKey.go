package utils

import "github.com/spf13/viper"

func EnvtKeyValue(key string) string {

	viper.SetConfigFile(".env")
	viper.ReadInConfig()
	value := []byte(viper.GetString(key))

	return string(value)
}
