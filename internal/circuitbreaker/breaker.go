package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements a simple circuit breaker pattern
type CircuitBreaker struct {
	mu                  sync.RWMutex
	state               State
	failureCount        int
	lastFailureTime     time.Time
	lastStateChangeTime time.Time

	// Configuration
	maxFailures    int
	cooldownPeriod time.Duration
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures    int           // Number of failures before opening circuit
	CooldownPeriod time.Duration // Time to wait before attempting half-open
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	if config.MaxFailures == 0 {
		config.MaxFailures = 3
	}
	if config.CooldownPeriod == 0 {
		config.CooldownPeriod = 30 * time.Second
	}

	return &CircuitBreaker{
		state:          StateClosed,
		maxFailures:    config.MaxFailures,
		cooldownPeriod: config.CooldownPeriod,
	}
}

// Execute runs the given function if the circuit is closed or half-open
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.canAttempt() {
		return ErrCircuitOpen
	}

	err := fn()
	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// canAttempt checks if a request can be attempted
func (cb *CircuitBreaker) canAttempt() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if cooldown period has passed
		if time.Since(cb.lastStateChangeTime) >= cb.cooldownPeriod {
			// Transition to half-open
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.maxFailures {
			cb.state = StateOpen
			cb.lastStateChangeTime = time.Now()
		}
	case StateHalfOpen:
		// Single failure in half-open state reopens circuit
		cb.state = StateOpen
		cb.lastStateChangeTime = time.Now()
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0

	switch cb.state {
	case StateOpen:
		// Transition to half-open after cooldown
		if time.Since(cb.lastStateChangeTime) >= cb.cooldownPeriod {
			cb.state = StateHalfOpen
		}
	case StateHalfOpen:
		// Success in half-open state closes the circuit
		cb.state = StateClosed
		cb.lastStateChangeTime = time.Now()
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStateValue returns numeric value for metrics (0=closed, 1=open, 2=half-open)
func (cb *CircuitBreaker) GetStateValue() int64 {
	state := cb.GetState()
	return int64(state)
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.lastStateChangeTime = time.Now()
}
