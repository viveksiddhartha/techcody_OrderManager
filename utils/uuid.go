package utils

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	idLength = 32
	idChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

var (
	idRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// NewID generates a new random ID
func NewID() string {
	b := make([]byte, idLength)
	for i := range b {
		b[i] = idChars[idRand.Intn(len(idChars))]
	}
	return fmt.Sprintf("%s-%d", string(b), time.Now().UnixNano())
}
