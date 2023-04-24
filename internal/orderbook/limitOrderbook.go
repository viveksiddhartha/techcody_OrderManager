package orderbook

import (
	"context"
	"errors"
	"fmt"
	"log"

	"mfus_OMV1/pkg/database"
	"mfus_OMV1/utils"
	"sort"
	"time"

	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// NewLimitOrder creates a new limit order
func NewLimitOrder(side string, quantity int64, price float64, timestamp time.Time) *LimitOrder {
	return &LimitOrder{
		OrderModel: OrderModel{Side: side, Quantity: quantity, Price: price},
		Timestamp:  timestamp,
	}
}

func StartLimitOrderMatch() {

	// To Do: Need to implement the Redis and MongoDB connection Client
	mongoClient, err := database.GetMongoClient()
	if err != nil {
		log.Fatal(err)
	}

	defer mongoClient.Disconnect(context.Background())

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	fmt.Print(redisClient)
	// Initialize order book
	orderBook := OrderBook{
		BuyOrders:  make([]OrderModel, 0),
		SellOrders: make([]OrderModel, 0),
	}

	fmt.Print(orderBook)
	// Start order matching loop
	for {
		fmt.Println("Run Go routine")
		// Get the latest buy and sell orders from MongoDB
		buyOrders, err := getOpenOrders("Buy", mongoClient)
		if err != nil {
			log.Printf("Error getting buy orders: %v", err)
			continue
		}
		fmt.Print(buyOrders)
		sellOrders, err := getOpenOrders("Sell", mongoClient)
		if err != nil {
			log.Printf("Error getting sell orders: %v", err)
			continue
		}
		fmt.Print(sellOrders)

		// Update order book with latest buy and sell orders
		orderBook.BuyOrders = buyOrders
		orderBook.SellOrders = sellOrders

		// Match orders
		matchOrders(&orderBook, redisClient, mongoClient)

		// Wait for a short interval before checking again
		time.Sleep(1 * time.Second)
	}

}

func getOpenOrders(side string, mongoClient *mongo.Client) ([]OrderModel, error) {
	// Connect to "orders" collection in MongoDB and get all open orders for the specified side (buy or sell)
	collection := mongoClient.Database("orderbook").Collection("orders")
	filter := bson.M{"side": side, "status": bson.M{"$ne": "cancelled"}}
	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	// Convert cursor to slice of Order objects
	orders := make([]OrderModel, 0)
	for cursor.Next(context.Background()) {
		var order OrderModel
		if err := cursor.Decode(&order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func matchOrders(orderBook *OrderBook, redisClient *redis.Client, mongoClient *mongo.Client) {
	// Sort buy and sell orders in order of highest to lowest price
	sort.Slice(orderBook.BuyOrders, func(i, j int) bool {
		return orderBook.BuyOrders[i].Price > orderBook.BuyOrders[j].Price
	})
	sort.Slice(orderBook.SellOrders, func(i, j int) bool {
		return orderBook.SellOrders[i].Price < orderBook.SellOrders[j].Price
	})

	// Match orders by iterating over buy and sell orders and executing trades
	for len(orderBook.BuyOrders) > 0 && len(orderBook.SellOrders) > 0 {
		buyOrder := orderBook.BuyOrders[0]
		sellOrder := orderBook.SellOrders[0]

		// Check if buy order price is greater than or equal to sell order price
		if buyOrder.Price >= sellOrder.Price {
			// Determine the quantity of the trade and update the orders accordingly
			var quantity int64
			if buyOrder.Quantity-buyOrder.FilledQty <= sellOrder.Quantity-sellOrder.FilledQty {
				quantity = buyOrder.Quantity - buyOrder.FilledQty
				buyOrder.FilledQty = buyOrder.Quantity
				sellOrder.FilledQty += quantity
				if sellOrder.FilledQty == sellOrder.Quantity {
					sellOrder.Status = "filled"
				} else {
					sellOrder.Status = "partially_filled"
				}
				orderBook.SellOrders[0] = sellOrder
				orderBook.SellOrders = orderBook.SellOrders[1:]
			} else {
				quantity = sellOrder.Quantity - sellOrder.FilledQty
				sellOrder.FilledQty = sellOrder.Quantity
				buyOrder.FilledQty += quantity
				if buyOrder.FilledQty == buyOrder.Quantity {
					buyOrder.Status = "filled"
				} else {
					buyOrder.Status = "partially_filled"
				}
				orderBook.BuyOrders[0] = buyOrder
				orderBook.BuyOrders = orderBook.BuyOrders[1:]
			}

			// Execute the trade by writing to Redis
			trade := map[string]interface{}{
				"buy_order_id":  buyOrder.ID,
				"sell_order_id": sellOrder.ID,
				"price":         sellOrder.Price,
				"quantity":      quantity,
				"timestamp":     time.Now().Unix(),
			}

			collection := mongoClient.Database("orderbook").Collection("trades")
			record, err := collection.InsertOne(context.Background(), trade)
			if err != nil {
				log.Printf("Error writing trade to Redis: %v", err)
			}

			fmt.Print(record)
			err = redisClient.LPush("trades", trade).Err()
			if err != nil {
				log.Printf("Error writing trade to Redis: %v", err)
			}
		} else {
			// No more orders to match
			break
		}
	}

	// Update buy and sell orders in MongoDB with new statuses and filled quantities
	updateOrders(orderBook.BuyOrders, mongoClient)
	updateOrders(orderBook.SellOrders, mongoClient)
}

func updateOrders(orders []OrderModel, mongoClient *mongo.Client) {
	// Connect to "orders" collection in MongoDB and update orders with new status and filled quantity
	collection := mongoClient.Database("orderbook").Collection("orders")
	for _, order := range orders {
		filter := bson.M{"id": order.ID}
		update := bson.M{"$set": bson.M{"status": order.Status, "filled": order.FilledQty}}
		_, err := collection.UpdateOne(context.Background(), filter, update)
		if err != nil {
			log.Printf("Error updating order %s: %v", order.ID, err)
		}
	}
}

func AddOrder(order OrderModel, mongoClient *mongo.Client) error {
	// Connect to "orders" collection in MongoDB and insert the order

	collection := mongoClient.Database("orderbook").Collection("orders")
	order.ID = primitive.ObjectID{}
	order.Status = "open"
	order.Expiration = utils.GetCurrentTimestamp()
	record, err := collection.InsertOne(context.Background(), order)
	if err != nil {
		return err
	}
	fmt.Println(record)
	return nil
}

func cancelOrder(id primitive.ObjectID, mongoClient *mongo.Client) error {
	// Connect to "orders" collection in MongoDB and update the order status to "cancelled"

	collection := mongoClient.Database("orderbook").Collection("orders")
	record, err := collection.UpdateMany(context.Background(),
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"status": "cancelled"}},
	)
	if err != nil {
		return err
	}
	fmt.Println(record)
	return nil
}

func handleOrder(order OrderModel, mongoClient *mongo.Client) error {
	// Validate the order
	err := validateOrder(order)
	if err != nil {
		return err
	}
	// Add the order to the database
	err = AddOrder(order, mongoClient)
	if err != nil {
		return err
	}

	return nil
}

func handleCancel(id string, mongoClient *mongo.Client) error {
	// Parse the order ID
	orderID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	// Cancel the order
	err = cancelOrder(orderID, mongoClient)
	if err != nil {
		return err
	}

	return nil
}

func handleCommand(command string, mongoClient *mongo.Client) error {
	switch command {
	case "exit":
		return errors.New("exit")
	case "help":
		fmt.Println("Available commands:")
		fmt.Println("buy <quantity> <price>")
		fmt.Println("sell <quantity> <price>")
		fmt.Println("cancel <order_id>")
		fmt.Println("exit")
	default:
		// Parse the command
		var order OrderModel
		var quantity int64
		var price float64
		n, err := fmt.Sscanf(command, "%s %f %f", &order.Side, &quantity, &price)
		if err != nil || n != 3 {
			return errors.New("invalid command")
		}
		order.Quantity = quantity
		order.Price = price
		// Handle the order
		err = handleOrder(order, mongoClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleInput(inputChan chan string, mongoClient *mongo.Client) {
	// Read commands from standard input and send them to the input channel
	for {
		var command string
		fmt.Scanln(&command)
		inputChan <- command
	}
}

func handleCommands(inputChan chan string, mongoClient *mongo.Client) {
	// Process commands from the input channel
	for {
		select {
		case command := <-inputChan:
			err := handleCommand(command, mongoClient)
			if err != nil {
				log.Printf("Error handling command %q: %v", command, err)
			}
		}
	}
}

func validateOrder(order OrderModel) error {
	if order.Type != "market" && order.Type != "limit" {
		return errors.New("invalid order type")
	}
	if order.Side != "buy" && order.Side != "sell" {
		return errors.New("invalid order side")
	}
	if order.Price <= 0 {
		return errors.New("invalid order price")
	}
	if order.Quantity <= 0 {
		return errors.New("invalid order quantity")
	}

	return nil
}
