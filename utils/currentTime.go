package utils

import "time"

// getTimestamp returns the current timestamp in milliseconds.
func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// getTimestamp returns the current timestamp as an integer.
func GetTimestampInteger() int64 {
	return time.Now().Unix()
}
