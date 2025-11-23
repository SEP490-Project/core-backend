package logging

import (
	"sync"

	"github.com/petermattis/goid"
)

var (
	// reqIDMap stores the mapping between Goroutine ID and Request ID
	reqIDMap sync.Map
)

// SetRequestID binds a request ID to the current goroutine
func SetRequestID(reqID string) {
	reqIDMap.Store(goid.Get(), reqID)
}

// GetRequestID retrieves the request ID for the current goroutine
func GetRequestID() string {
	if val, ok := reqIDMap.Load(goid.Get()); ok {
		return val.(string)
	}
	return ""
}

// CleanupRequestID removes the binding. Crucial for memory management.
func CleanupRequestID() {
	reqIDMap.Delete(goid.Get())
}
