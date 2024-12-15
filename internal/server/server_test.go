package server

import (
	"testing"
)

func TestServer(t *testing.T) {
	server := New(&Config{
		Host: "0.0.0.0",
		Port: "8000",
	})
	server.Run()
}
