package orderbook

import (
	"context"
	"encoding/json"
	"mfus_OMV1/pkg/database"
	"mfus_OMV1/utils"

	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Define the database and collection names
var dbName = "orderbook"
var ordersCollection = "orders"
var tradeCollection = "trades"
var stateChangesCollection = "orders_state"

type OrderHandlers interface {
}

func MongoDBOrderBookCollectionFunc() (*mongo.Client, error) {
	db, err := database.GetMongoClient()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Parse order from request body
	var order OrderModel
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set default values for order
	order.Status = "Open"
	order.CreationTime = utils.GetCurrentTimestamp()
	order.UpdateTime = utils.GetCurrentTimestamp()

	client, err := database.GetMongoClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := client.Database(dbName).Collection(ordersCollection).InsertOne(context.Background(), order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add initial state change
	stateChange := StateChange{
		OrderID:   result.InsertedID.(primitive.ObjectID),
		FromState: "",
		ToState:   string(Open),
		CreatedAt: time.Now(),
	}
	_, err = client.Database(dbName).Collection(stateChangesCollection).InsertOne(context.Background(), stateChange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return created order
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	// Get all orders from MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := database.GetMongoClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cursor, err := client.Database(dbName).Collection(ordersCollection).Find(ctx, bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)
	// Decode orders from cursor
	var orders []OrderModel
	for cursor.Next(ctx) {
		var order OrderModel

		err := cursor.Decode(&order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}
	if err := cursor.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return orders
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)

}

func GetOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Get order ID from URL parameters
	params := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get order from MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var order OrderModel

	client, err := database.GetMongoClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = client.Database(dbName).Collection(ordersCollection).FindOne(ctx, bson.M{"_id": id}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Order not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return order
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}

func UpdateOpenOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Get order ID from URL parameters
	params := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update order status to "cancelled"
	filter := bson.M{"_id": id, "status": "open"}

	// Parse order from request body
	var orderUpdates bson.M
	err = json.NewDecoder(r.Body).Decode(&orderUpdates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set updated_at timestamp
	orderUpdates["updated_at"] = time.Now()

	// Update order in MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := database.GetMongoClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := client.Database(dbName).Collection(ordersCollection).UpdateOne(ctx, filter, bson.M{"$set": orderUpdates})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result.ModifiedCount == 0 {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Get current state of order
	var order OrderModel
	err = client.Database(dbName).Collection(ordersCollection).FindOne(ctx, bson.M{"_id": id}).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if state has changed
	currentState := order.Status
	newState := orderUpdates["Status"]
	if currentState != newState {
		// Add state change to state changes collection
		stateChange := StateChange{
			OrderID:   id,
			FromState: currentState,
			ToState:   newState.(string),
			CreatedAt: time.Now(),
		}
		_, err = client.Database(dbName).Collection(stateChangesCollection).InsertOne(ctx, stateChange)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Return updated order
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orderUpdates)
}

func GetOpenOrdersHandler(w http.ResponseWriter, r *http.Request) {
	var order OrderModel
	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	filter := bson.M{
		"symbol": order.Symbol,
		"side":   order.Side,
		"status": "open",
	}
	client, err := database.GetMongoClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cursor, err := client.Database(dbName).Collection(ordersCollection).Find(context.Background(), filter)
	if err != nil {
		return
	}
	var orders []OrderModel
	if err = cursor.All(context.Background(), &orders); err != nil {
		return
	}

	// Return order
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}

func DeleteOrderHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, _ := primitive.ObjectIDFromHex(params["id"])

	client, err := database.GetMongoClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := client.Database(dbName).Collection(ordersCollection).DeleteOne(context.Background(), bson.M{"_id": id, "status": "open"})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result.DeletedCount)
}

// To Do: Add the CancelOrder function here to handle the In Memory
func CancelOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get order ID from URL parameters
	params := mux.Vars(r)
	orderID := params["id"]

	// Update order status to "cancelled"
	filter := bson.M{"_id": orderID, "status": "open"}
	update := bson.M{"$set": bson.M{"status": "cancelled"}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := database.GetMongoClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := client.Database(dbName).Collection(ordersCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if result.ModifiedCount == 0 {
		http.Error(w, "Order not found or already closed", http.StatusNotFound)
		return
	}

	// Return the updated order
	filter = bson.M{"_id": orderID}
	var order OrderModel
	err = client.Database(dbName).Collection(ordersCollection).FindOne(ctx, filter).Decode(&order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(order)
}
