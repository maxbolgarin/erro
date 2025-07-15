package erro_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/maxbolgarin/erro"
)

func TestList(t *testing.T) {
	list := erro.NewList()
	list.Add(errors.New("err1"))
	list.New("err2", "key", "value")
	list.Wrap(errors.New("err3"), "err4")

	if list.Len() != 3 {
		t.Fatalf("Expected list length 3, got %d", list.Len())
	}

	err := list.Err()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "err1") {
		t.Errorf("Expected error to contain 'err1'")
	}
	if !strings.Contains(err.Error(), "err2") {
		t.Errorf("Expected error to contain 'err2'")
	}
	if !strings.Contains(err.Error(), "err4") {
		t.Errorf("Expected error to contain 'err4'")
	}

	list.Remove(1)
	if list.Len() != 2 {
		t.Fatalf("Expected list length 2, got %d", list.Len())
	}

	list.Clear()
	if list.Len() != 0 {
		t.Fatalf("Expected list length 0, got %d", list.Len())
	}
}

func TestSet(t *testing.T) {
	set := erro.NewSet()
	set.Add(errors.New("err1"))
	set.Add(errors.New("err1"))
	set.New("err2")
	set.New("err2")

	if set.Len() != 2 {
		t.Fatalf("Expected set length 2, got %d", set.Len())
	}

	err := set.Err()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "[2 times]") {
		t.Errorf("Expected error to contain '[2 times]'")
	}
}

func TestSafeList(t *testing.T) {
	safeList := erro.NewSafeList()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			safeList.New("test error")
		}()
	}
	wg.Wait()

	if safeList.Len() != 100 {
		t.Errorf("Expected safe list length 100, got %d", safeList.Len())
	}
}

func TestSafeSet(t *testing.T) {
	safeSet := erro.NewSafeSet()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			safeSet.New("test error")
		}()
	}
	wg.Wait()

	if safeSet.Len() != 1 {
		t.Errorf("Expected safe set length 1, got %d", safeSet.Len())
	}
}
