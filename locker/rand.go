package locker

import (
	"math/rand"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().Unix()))

func randomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[r.Intn(len(letter))]
	}

	return string(b)
}
