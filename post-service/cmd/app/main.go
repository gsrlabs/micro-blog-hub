package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/cache"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/db"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/config"
)

const configPath = "config/config.yml"

func main() {

	

	log.Printf("INFO: starting application")

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config failed: %v", err)
		//return err
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config validate failed: %v", err)
		//return err
	}

	ctx := context.Background()

	//Mongo
	mongoClient, err := db.NewMongoCLient(ctx, cfg.Mongo.Host, cfg.Mongo.Port, cfg.Mongo.DB)

	if err != nil {
		log.Fatalf("Mongo connection failed: %v", err)
	}

	defer mongoClient.Disconnect(ctx)

	log.Println("Conected to mongo")

	//Redis
	redisClient, err := cache.NewRedisClient(ctx, cfg.Redis.Host, cfg.Redis.Port)
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	defer redisClient.Close()

	log.Println("Connectinon to Redis")

	//HTTP
	r := gin.Default()
r.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})

	log.Fatal(r.Run(":" + cfg.App.Port))

}

