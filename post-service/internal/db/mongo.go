package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewMongoCLient(parent context.Context, host string, port string, ) (*mongo.Client, error) {

	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
    defer cancel()

	uri := fmt.Sprintf("mongodb://%s:%s", host, port)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil{
		return nil, err
	}

	if err := client.Ping(ctx, nil); err !=nil {
		return nil, err
	}


	return client, nil

}