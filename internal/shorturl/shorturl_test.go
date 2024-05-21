package shorturl

import (
	"math/rand"
	"testing"
	"time"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func BenchmarkGenerateLen10(b *testing.B) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randStr := randString(10)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Generate(randStr); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateLen100(b *testing.B) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randStr := randString(100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Generate(randStr); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateLen1000(b *testing.B) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randStr := randString(1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Generate(randStr); err != nil {
			b.Fatal(err)
		}
	}
}

func randString(n int) string {
	b := make([]byte, n)
	ln := len(alphabet)
	for i := range b {
		b[i] = alphabet[rand.Intn(ln)]
	}
	return string(b)
}
