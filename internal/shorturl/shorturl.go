// Package shorturl manages the conversion of the original URL to a short one.
package shorturl

import (
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/itchyny/base58-go"
	"github.com/pkg/errors"
)

// sha256of hashes the initial input using standart sha256 hash function
func sha256Of(input []byte) []byte {
	algorithm := sha256.New()
	algorithm.Write(input)
	return algorithm.Sum(nil)
}

// base58Encoded utilizes Base58 algorithm instead of standard base-64 encoding
// Base58 reduces confusion in character output (0OIl+/ are not used).
func base58Encoded(bytes []byte) (string, error) {
	encoding := base58.BitcoinEncoding
	encoded, err := encoding.Encode(bytes)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// Generate produces a short link with a length of 8 characters from the original one
func Generate(initialLink string) (string, error) {
	urlHashBytes := sha256Of([]byte(initialLink))
	generatedNumber := new(big.Int).SetBytes(urlHashBytes).Uint64()
	finalString, err := base58Encoded([]byte(fmt.Sprintf("%d", generatedNumber)))
	if err != nil {
		return "", errors.Wrap(err, "failed to generate short link")
	}
	return finalString[:8], nil
}
