package config_test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/stretchr/testify/require"
)

func ExampleNetAddress_String() {
	addr := &config.NetAddress{Host: "example.com", Port: 8080}
	fmt.Println(addr.String()) // Output: example.com:8080
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

func ExampleDuration_Set() {
	d := new(config.Duration)

	err := d.Set("2h45m")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(d.String()) // Output: 2h45m0s
}

func ExampleDuration_String() {
	d := config.Duration(time.Hour * 3)
	fmt.Println(d.String()) // Output: 3h0m0s
}

func TestDuration_SetInvalid(t *testing.T) {
	d := new(config.Duration)

	err := d.Set("invalid")
	require.Error(t, err, "invalid duration produces no error")
}

func ExampleFileStorage_Set_noDefault() {
	fs := config.NewFileStorage()

	err := fs.Set("path/to/storage")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(fs.String(), fs.WriteRequired())
	// Output: path/to/storage true
}

func ExampleFileStorage_Set_default() {
	fs := config.NewFileStorage()

	err := fs.Set("")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(fs.String(), fs.WriteRequired())
	// Output: /tmp/short-url-db.json false
}

func TestFileStorage_Path(t *testing.T) {
	fs := config.NewFileStorage()

	path := "path/to/storage"

	err := fs.Set(path)
	require.NoError(t, err)

	require.Equal(t, path, fs.Path())
	require.True(t, fs.WriteRequired())
}
