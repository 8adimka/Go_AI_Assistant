package circuitbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/circuitbreaker"
)

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 100 * time.Millisecond,
	})

	// Circuit should start closed
	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("Expected state Closed, got %v", cb.GetState())
	}

	// Successful execution should keep circuit closed
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("Expected state Closed after success, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_OpenAfterFailures(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 100 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// First 2 failures should keep circuit closed
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return testErr
		})
		if err != testErr {
			t.Errorf("Expected test error, got %v", err)
		}
		if cb.GetState() != circuitbreaker.StateClosed {
			t.Errorf("Expected state Closed after %d failures, got %v", i+1, cb.GetState())
		}
	}

	// Third failure should open the circuit
	err := cb.Execute(func() error {
		return testErr
	})
	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}

	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("Expected state Open after 3 failures, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_RejectsWhenOpen(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 100 * time.Millisecond,
	})

	// Force circuit open by triggering failures
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb.Execute(func() error { return testErr })
	}

	// Circuit should be open
	if cb.GetState() != circuitbreaker.StateOpen {
		t.Fatalf("Expected state Open, got %v", cb.GetState())
	}

	// Attempt should be rejected immediately
	err := cb.Execute(func() error {
		t.Error("Function should not be called when circuit is open")
		return nil
	})

	if err != circuitbreaker.ErrCircuitOpen {
		t.Errorf("Expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 50 * time.Millisecond,
	})

	// Force circuit open
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb.Execute(func() error { return testErr })
	}

	if cb.GetState() != circuitbreaker.StateOpen {
		t.Fatalf("Expected state Open, got %v", cb.GetState())
	}

	// Wait for cooldown period
	time.Sleep(60 * time.Millisecond)

	// Next attempt should be allowed (circuit transitions to half-open)
	executed := false
	err := cb.Execute(func() error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected successful execution after cooldown, got error: %v", err)
	}

	if !executed {
		t.Error("Function was not executed after cooldown")
	}

	// After successful execution in half-open, should transition to closed
	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("Expected state Closed after successful half-open attempt, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_FailureInHalfOpenReopens(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 50 * time.Millisecond,
	})

	// Force circuit open
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb.Execute(func() error { return testErr })
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Attempt should be allowed but fail
	err := cb.Execute(func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}

	// Circuit should be open again
	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("Expected state Open after failure in half-open, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 100 * time.Millisecond,
	})

	testErr := errors.New("test error")

	// 2 failures
	for i := 0; i < 2; i++ {
		cb.Execute(func() error { return testErr })
	}

	// 1 success - should reset count
	cb.Execute(func() error { return nil })

	// Circuit should still be closed
	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("Expected state Closed, got %v", cb.GetState())
	}

	// Now 3 more failures should be needed to open
	for i := 0; i < 2; i++ {
		cb.Execute(func() error { return testErr })
		if cb.GetState() != circuitbreaker.StateClosed {
			t.Errorf("Expected state Closed, got %v", cb.GetState())
		}
	}

	// Third failure should open
	cb.Execute(func() error { return testErr })
	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("Expected state Open, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_GetStateValue(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 100 * time.Millisecond,
	})

	// Closed = 0
	if cb.GetStateValue() != 0 {
		t.Errorf("Expected state value 0 for Closed, got %d", cb.GetStateValue())
	}

	// Force open - Open = 1
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb.Execute(func() error { return testErr })
	}
	if cb.GetStateValue() != 1 {
		t.Errorf("Expected state value 1 for Open, got %d", cb.GetStateValue())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    3,
		CooldownPeriod: 100 * time.Millisecond,
	})

	// Force circuit open
	testErr := errors.New("test error")
	for i := 0; i < 3; i++ {
		cb.Execute(func() error { return testErr })
	}

	if cb.GetState() != circuitbreaker.StateOpen {
		t.Fatalf("Expected state Open, got %v", cb.GetState())
	}

	// Reset should close the circuit
	cb.Reset()

	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("Expected state Closed after reset, got %v", cb.GetState())
	}

	// Should be able to execute immediately
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Expected successful execution after reset, got error: %v", err)
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{})

	// Should use defaults: 3 failures, 30s cooldown
	testErr := errors.New("test error")

	// 2 failures should keep circuit closed
	for i := 0; i < 2; i++ {
		cb.Execute(func() error { return testErr })
		if cb.GetState() != circuitbreaker.StateClosed {
			t.Errorf("Expected state Closed, got %v", cb.GetState())
		}
	}

	// 3rd failure should open
	cb.Execute(func() error { return testErr })
	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("Expected state Open after 3 failures, got %v", cb.GetState())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		MaxFailures:    5,
		CooldownPeriod: 100 * time.Millisecond,
	})

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cb.Execute(func() error {
					if j%3 == 0 {
						return errors.New("test error")
					}
					return nil
				})
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic - just verify we can still get state
	_ = cb.GetState()
}
