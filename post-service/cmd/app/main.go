package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/cache"
	"github.com/gsrlabs/micro-blog-hub/post-service/internal/db"
)

func main() {

	ctx := context.Background()

	mongoHost := os.Getenv("MONGO_HOST")
	mongoPort := os.Getenv("MONGO_PORT")

	redisHost := os.Getenv("REDIS_HOST")
	residPost := os.Getenv("REDIS_PORT")

	//Mongo

	mongoClient, err := db.NewMongoCLient(ctx, mongoHost, mongoPort)

	if err != nil {
		log.Fatalf("Mongo connection failed: %v", err)
	}

	defer mongoClient.Disconnect(nil)

	log.Println("Conected to mongo")

	//Redis

	redisClient, err := cache.NewRedisClient(ctx, redisHost, residPost)
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}

	defer redisClient.Close()

	log.Println("Connectinon to Redis")

	//HTTP

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	r.Run(":8050")

}

