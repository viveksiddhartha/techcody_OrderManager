package orderbook

type OrderType int

const (
	Market OrderType = iota
	Limit
)

type Side int

const (
	Buy Side = iota
	Sell
)

type OrderStatus int

const (
	Open OrderStatus = iota
	Partial
	Cancelled
	Filled
)
