package utils

import (
	"crypto/rand"
	"math/big"
)

func RandomString(len int) string {
	var letters = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
	ret := make([]rune, len)
	var randInt *big.Int
	for i := range ret {
		randInt, _ = rand.Int(rand.Reader, big.NewInt(26+26+10))
		ret[i] = letters[randInt.Int64()]
	}
	return string(ret)
}
