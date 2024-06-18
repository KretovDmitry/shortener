package shorturl

import (
	"math/rand"
	"regexp"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func BenchmarkGenerateLen10(b *testing.B) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randStr := randString(10)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Generate(randStr)
	}
}

func BenchmarkGenerateLen100(b *testing.B) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randStr := randString(100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Generate(randStr)
	}
}

func BenchmarkGenerateLen1000(b *testing.B) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randStr := randString(1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Generate(randStr)
	}
}

func randString(length uint) string {
	b := make([]byte, length)
	ln := len(alphabet)
	for i := range b {
		b[i] = alphabet[rand.Intn(ln)]
	}
	return string(b)
}

func FuzzGenerate(f *testing.F) {
	// base58Regexp is a regular expression that matches
	// a valid Base58-encoded string of any length.
	base58Regexp := regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]+$`)

	testcases := []string{
		"https://go.dev",
		"https://yandex.ru",
		"https://google.com",
		"https://github.com",
		"https://medium.com",
	}
	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, a string) {
		res := Generate(a)
		assert.True(t, utf8.ValidString(res), "invalid utf-8 sequence")
		assert.True(t, base58Regexp.MatchString(res),
			"generated string expected to be base58 encoded")
	})
}
