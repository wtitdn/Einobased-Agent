package main

import (
	"context"
	"log"
	"os"
	"strconv"

	agents "einoproject/internal/Agents"
	"einoproject/internal/config"
	controller "einoproject/internal/controller"
	"einoproject/internal/db"
	pkgredis "einoproject/pkg/redis"
	"einoproject/pkg/server"
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
	//Connect db
	// Initialize database and run migrations.
	sqlDB, err := db.NewDB(cfg.Db)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if err := db.AutoMigrate(sqlDB); err != nil {
		log.Fatalf("failed to auto migrate database: %v", err)
	}
	defer db.CloseDB(sqlDB)
	// connet redis
	  
	redisClient, err := pkgredis.NewClient(context.Background(), cfg.Redis)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer redisClient.Close()

	registeredAgents := agents.RegisterAgents(context.Background())
	r := controller.SetRouter(registeredAgents, sqlDB, redisClient)
	server.RunGracefully(":"+strconv.Itoa(cfg.Server.Port), r)
}
