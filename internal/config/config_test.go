package config_test

import (
	"fmt"
	"log"
	"testing"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/stretchr/testify/require"
)

func ExampleNetAddress_String() {
	addr := config.NewNetAddress()
	fmt.Println(addr.String()) // Output: 0.0.0.0:8080
}

func ExampleNetAddress_Set() {
	addr := config.NewNetAddress()

	err := addr.Set("example.com:8080")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(addr.String()) // Output: example.com:8080
}

func TestNetAddress_SetInvalid(t *testing.T) {
	addr := config.NewNetAddress()

	cases := []struct {
		input string
	}{
		{input: "invalid"},
		{input: "example.com"},
		{input: "example.com:NaN"},
		{input: "example.com:8080:8080"},
		{input: "example.com:8080:8080:8080"},
	}

	for _, c := range cases {
		err := addr.Set(c.input)
		require.Error(t, err, "invalid address produces no error")
	}
}
