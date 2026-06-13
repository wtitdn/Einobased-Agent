package main

import (
	"os"
)

func main() {

	// Load application config.
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config/config.yaml"
	}

}
