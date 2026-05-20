package controller

import (
	"testing"
)

func TestPathID(t *testing.T) {
	id, err := pathID("/v1/profiles/42/enabled", "/enabled")
	if err != nil || id != 42 {
		t.Fatalf("got %d err=%v", id, err)
	}
	id, err = pathID("/v1/profiles/7/connect", "/connect")
	if err != nil || id != 7 {
		t.Fatalf("connect: got %d err=%v", id, err)
	}
}
