package gzipfork

// this is a straight copy of github.com/elephant-insurance/go-microservice-arch/v3/clicker/clicker.go
// to avoid an import cycle
// TODO: move clicker to go-base-packages

import (
	"strconv"
	"sync"
)

// Clicker is a very simple thread-safe counter
// Use it for updating counts of events for logging and diagnostics
type Clicker interface {
	// Click increments the counter by one and returns the new count
	Click() int
	// Count returns the current count of clicks
	Count() int
	// Add adds a specified number of clicks to the counter and returns the new count
	Add(count int) int
	// Set sets the click count to a specified value and returns the clicker
	Set(count int) Clicker
	// String returns the click count as a string for easy display
	String() string
}

func newClicker() Clicker {
	return &clickerType{}
}

type clickerType struct {
	clickCount int
	sync.RWMutex
}

// Add adds count clicks to the counter and returns the result
func (c *clickerType) Add(count int) int {
	if count != 0 {
		c.Lock()
		defer c.Unlock()
		c.clickCount += count
	}

	return c.clickCount
}

// Click adds one click to the counter and returns the result
func (c *clickerType) Click() int {
	c.Lock()
	defer c.Unlock()
	c.clickCount++

	return c.clickCount
}

// Count returns the current count of clicks
func (c *clickerType) Count() int {
	c.RLock()
	defer c.RUnlock()
	return c.clickCount
}

// Set sets the clickerType's clickCount to a specified int
func (c *clickerType) Set(count int) Clicker {
	if count != c.clickCount {
		c.Lock()
		defer c.Unlock()
		c.clickCount = count
	}

	return c
}

// String returns the click count as a string
func (c *clickerType) String() string {
	c.RLock()
	defer c.RUnlock()
	return strconv.Itoa(c.clickCount)
}

// MarshalJSON lets us to output a clickerType struct as if it were a scalar int
func (c *clickerType) MarshalJSON() ([]byte, error) {
	return []byte(c.String()), nil
}
