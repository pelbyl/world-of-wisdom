package wisdom

import (
	"sync"
	"testing"
)

func TestNewQuoteProvider(t *testing.T) {
	qp := NewQuoteProvider()
	
	if qp == nil {
		t.Fatal("NewQuoteProvider returned nil")
	}
	
	if qp.GetQuoteCount() == 0 {
		t.Error("Quote provider should have default quotes")
	}
}

func TestGetRandomQuote(t *testing.T) {
	qp := NewQuoteProvider()
	
	quotesMap := make(map[string]bool)
	
	for i := 0; i < 100; i++ {
		quote := qp.GetRandomQuote()
		if quote == "" {
			t.Error("GetRandomQuote returned empty string")
		}
		quotesMap[quote] = true
	}
	
	if len(quotesMap) < 5 {
		t.Errorf("GetRandomQuote returned only %d unique quotes out of 100 calls", len(quotesMap))
	}
}

func TestAddQuote(t *testing.T) {
	qp := NewQuoteProvider()
	initialCount := qp.GetQuoteCount()
	
	testQuote := "Test wisdom quote"
	qp.AddQuote(testQuote)
	
	if qp.GetQuoteCount() != initialCount+1 {
		t.Errorf("Quote count should be %d, got %d", initialCount+1, qp.GetQuoteCount())
	}
	
	found := false
	for i := 0; i < 1000 && !found; i++ {
		if qp.GetRandomQuote() == testQuote {
			found = true
		}
	}
	
	if !found {
		t.Error("Added quote was not returned by GetRandomQuote")
	}
}

func TestConcurrentAccess(t *testing.T) {
	qp := NewQuoteProvider()
	var wg sync.WaitGroup
	
	numGoroutines := 100
	numOperations := 100
	
	wg.Add(numGoroutines * 2)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = qp.GetRandomQuote()
			}
		}(i)
		
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				qp.AddQuote("Concurrent quote " + string(rune(id)))
			}
		}(i)
	}
	
	wg.Wait()
	
	finalCount := qp.GetQuoteCount()
	expectedCount := len(quotes) + (numGoroutines * numOperations)
	
	if finalCount != expectedCount {
		t.Errorf("Expected %d quotes after concurrent operations, got %d", expectedCount, finalCount)
	}
}