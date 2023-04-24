package orderbook

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrderModel struct {
	ID            primitive.ObjectID    `json:"id" bson:"_id"`
	UserID        string                `json:"userID" bson:"userID"`
	Symbol        string                `json:"symbol" bson:"symbol"`
	Price         float64               `json:"price" bson:"price"`
	Quantity      int64                 `json:"quantity" bson:"quantity"`
	RemainingQty  int64                 `json:"remainingQty" bson:"remainingQty"`
	Side          string                `json:"side" bson:"side"`
	Type          string                `json:"type" bson:"type"`
	Status        string                `json:"status" bson:"status"`
	Expiration    int64                 `json:"expiration" bson:"expiration"`
	CreationTime  int64                 `json:"creationTime" bson:"creationTime"`
	UpdateTime    int64                 `json:"updateTime" bson:"updateTime"`
	FilledOrders  []string              `json:"filledOrders" bson:"filledOrders"`
	FilledQty     int64                 `json:"filledQty" bson:"filledQty"`
	FilledVolume  float64               `json:"filledVolume" bson:"filledVolume"`
	FilledAverage float64               `json:"filledAverage" bson:"filledAverage"`
	FilledOrder   *TradeFilledInfoModel `json:"filled_order" bson:"filled_order,omitempty"`
}

type OrderManagerModel struct {
	Symbol           string
	Collection       *mongo.Collection
	TradeCollection  *mongo.Collection
	Interval         time.Duration
	MaxTPS           int
	Ctx              context.Context
	Cancel           context.CancelFunc
	Orders           map[string]*OrderModel
	BuyOrders        []*OrderModel `json:"buyOrders"`
	SellOrders       []*OrderModel `json:"sellOrders"`
	FilledOrders     []*OrderModel `json:"filledOrders"`
	OrderBookModel   OrderBookModel
	tradeChan        chan *TradeHistoryModel
	orderChan        chan *OrderModel
	stopChan         chan struct{}
	OrderMutex       sync.Mutex
	TradeMutex       sync.Mutex
	LastTradeID      string
	LastTradePrice   float64
	LastTradeTime    time.Time
	TradeCount       int
	TotalTradeVolume float64
	orderMatchTicker *time.Ticker
	RedisClient      *redis.Client
	MongoClient      *mongo.Client
	mutex            sync.Mutex
	MarketPrice      float64
}

// OrderBook represents the order book
type OrderBookModel struct {
	Bids    *list.List
	Asks    *list.List
	Trades  *list.List
	Market  *MarketOrder
	updated time.Time
}

type TradeHistoryModel struct {
	Id         string
	BuyOrder   string
	SellOrder  string
	Quantity   int64
	Price      float64
	ExecutedAt time.Time
	Timestamp  time.Time
}

type TradeBookModel struct {
	ID        string     `json:"id"`
	BuyOrder  OrderModel `json:"buyOrder"`
	SellOrder OrderModel `json:"sellOrder"`
	Quantity  int64      `json:"quantity"`
	Price     float64    `json:"price"`
	Time      int64      `json:"time"`
}

type TradeFilledInfoModel struct {
	OrderID   string  `bson:"_id,omitempty"`
	Price     float64 `bson:"price"`
	Quantity  float64 `bson:"quantity"`
	Timestamp int64   `bson:"timestamp"`
}

// StateChange represents a change in order status
type StateChange struct {
	ID        primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	OrderID   primitive.ObjectID `json:"order_id,omitempty" bson:"order_id,omitempty"`
	FromState string             `json:"from_state,omitempty" bson:"from_state,omitempty"`
	ToState   string             `json:"to_state,omitempty" bson:"to_state,omitempty"`
	CreatedAt time.Time          `json:"created_at,omitempty" bson:"created_at,omitempty"`
}

// LimitOrder represents a limit order with time priority
type LimitOrder struct {
	OrderModel
	Timestamp time.Time
}

// MarketOrder represents a market order
type MarketOrder struct {
	OrderModel
}

type OrderBook struct {
	BuyOrders  []OrderModel
	SellOrders []OrderModel
}
