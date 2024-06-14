package shorturl

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/itchyny/base58-go"
)

// Generate produces a short link from the original one.
// It utilizes base 58 algorithm to reduce confusion in character output
// (0OIl+/ are not used).
func Generate(s string) string {
	sha256 := sha256.New()
	sha256.Write([]byte(s))
	urlHashBytes := sha256.Sum(nil)
	generatedNumber := binary.BigEndian.Uint64(urlHashBytes)
	encodedBytes := base58.BitcoinEncoding.EncodeUint64(generatedNumber)
	fmt.Println(string(encodedBytes))
	return string(encodedBytes)
}
