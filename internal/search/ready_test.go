package search

import (
	"testing"
	"time"
)

func Test_Ready_NilService(t *testing.T) {
	var s *Service
	if s.Ready() {
		t.Fatal("expected nil service to not be ready")
	}
}

func Test_Ready_EmptyIndex(t *testing.T) {
	s := &Service{}
	if s.Ready() {
		t.Fatal("expected empty service to not be ready")
	}
}

func Test_Ready_NilIndex(t *testing.T) {
	s := &Service{index: nil}
	if s.Ready() {
		t.Fatal("expected nil index to not be ready")
	}
}

func Test_Ready_PopulatedIndex(t *testing.T) {
	s := newTestService([]schematicIndex{
		{ID: "1", Title: "Farm", Created: time.Now()},
	})
	if !s.Ready() {
		t.Fatal("expected populated service to be ready")
	}
}
