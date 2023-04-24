package main

import (
	"log"
	"mfus_OMV1/internal/orderbook"
	"mfus_OMV1/pkg/database"
)

func main() {

	// create MongoDB client
	RedisClient, err := database.NewRedisClient()
	if err != nil {
		log.Fatal(err)
	}
	defer RedisClient.Close()

	orderbook.StartLimitOrderMatch()

}
