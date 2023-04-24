package database

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DBConn is a MongoDB connection struct
type DBConn struct {
	Client     *mongo.Client
	Collection *mongo.Collection
}

var mongoOnce sync.Once
var mongoClient *mongo.Client

func GetMongoClient() (*mongo.Client, error) {
	var err error
	mongoOnce.Do(func() {
		mongoClientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
		mongoClientOptions.SetMaxPoolSize(100)
		mongoClient, err = mongo.NewClient(mongoClientOptions)
		if err != nil {
			log.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = mongoClient.Connect(ctx)
		if err != nil {
			log.Fatal(err)
		}
	})
	return mongoClient, err
}

// To Do: Need to remove this method after verifing the  NewRedisClient
/*
func RedisClient(key string, value string) {
	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := redisFunction(key, value)
			if err != nil {
				fmt.Println("Error:", err)
			}
		}()
	}
	wg.Wait()

}
 func redisFunction(key string, value string) error {
	redisPool := &redis.Pool{
		MaxIdle:     1000,
		MaxActive:   1000,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
	}

	conn := redisPool.Get()
	defer conn.Close()

	_, err := conn.Do("SET", key, value)
	if err != nil {
		return err
	}

	return nil
} */

func NewRedisClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// test connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return client, nil
}
