package main

import (
	"fmt"

	"github.com/spf13/viper"
)

type config struct {
	urls             []string
	minTimeout       uint
	maxTimeout       uint
	numberOfRequests uint
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error of read config file: %+v", err))
	}

	conf := config{
		urls:             viper.GetStringSlice("URLs"),
		minTimeout:       viper.GetUint("MinTimeout"),
		maxTimeout:       viper.GetUint("MaxTimeout"),
		numberOfRequests: viper.GetUint("NumberOfRequests"),
	}
	fmt.Printf("config = %+v\n", conf)
}
