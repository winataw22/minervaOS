package tnodb

import (
	"testing"
)

func TestIFaceToPublish(t *testing.T) {
	tnodb := NewHTTPHTTPTNoDB("http://172.20.0.1:8080")
	tnodb.PublishInterfaces()
}
