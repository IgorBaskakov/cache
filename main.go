package main

import (
	"fmt"

	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error of read config file: %+v", err))
	}
	fmt.Printf("url = %#v\n", viper.GetStringSlice("URLs"))
	fmt.Printf("MinTimeout = %#v\n", viper.GetInt("MinTimeout"))
	fmt.Printf("MaxTimeout = %#v\n", viper.GetInt("MaxTimeout"))
}
