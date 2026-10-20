package docker

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	// Attempting to connect to an unreachable remote host should error on Ping
	_, err := NewClient("tcp://127.0.0.1:59999")
	if err == nil {
		t.Error("Expected error when connecting to non-existent Docker host, got nil")
	}
}

func TestClient_Close(t *testing.T) {
	c := &Client{cli: nil}
	if err := c.Close(); err != nil {
		t.Errorf("Client.Close() on nil cli returned error: %v", err)
	}
}
