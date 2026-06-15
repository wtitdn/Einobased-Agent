package main

import (
	"context"
	"os"
	"strconv"

	agents "einoproject/internal/Agents"
	"einoproject/internal/config"
	controller "einoproject/internal/controller"
	"einoproject/internal/server"
)

func main() {

	// Load application config.
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "../config/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		panic(err)
	}

	registeredAgents := agents.RegisterAgents(context.Background())
	r := controller.SetRouter(registeredAgents)
	server.RunGracefully(":"+strconv.Itoa(cfg.Server.Port), r)
}
