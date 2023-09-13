package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Host        string
	ARIURL      string
	WSURL       string
	UseProxy    bool
	Username    string
	Password    string
	Application string
}

func NewConfig() *Config {

	host := os.Getenv("ARI_HOST")
	ariURL := fmt.Sprintf("http://%s:8088/ari", host)
	wsURL := fmt.Sprintf("ws://%s:8088/ari/events", host)

	return &Config{
		Host:        host,
		ARIURL:      ariURL,
		WSURL:       wsURL,
		UseProxy:    useProxy(),
		Username:    os.Getenv("ARI_USERNAME"),
		Password:    os.Getenv("ARI_PASSWORD"),
		Application: "lineblocs",
	}
}

func useProxy() bool {
	useProxy, err := strconv.ParseBool(os.Getenv("ARI_USE_PROXY"))
	if err != nil {
		return false
	}
	return useProxy
}
