package godaikin

import (
	"sync"
	"time"
)

// Values represents a smart container for appliance's data
// It keeps track of which values have been accessed and when resources were last updated
type Values struct {
	mu                   sync.RWMutex
	data                 map[string]string
	lastUpdateByResource map[string]time.Time
	resourceByKey        map[string]string
	ttl                  time.Duration
}

func NewValues() *Values {
	return &Values{
		data:                 make(map[string]string),
		lastUpdateByResource: make(map[string]time.Time),
		resourceByKey:        make(map[string]string),
		ttl:                  15 * time.Minute, // TTL for resource updates
	}
}

func (v *Values) Get(key string) (string, bool) {
	return v.GetWithInvalidation(key, true)
}

// GetWithInvalidation returns the value for the given key
// If invalidate is true, it marks the associated resource as needing update
func (v *Values) GetWithInvalidation(key string, invalidate bool) (string, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()

	value, exists := v.data[key]
	if !exists {
		return "", false
	}

	// If invalidate is true, mark the resource as needing update
	if invalidate {
		if resource, exists := v.resourceByKey[key]; exists {
			delete(v.lastUpdateByResource, resource)
		}
	}

	return value, true
}

func (v *Values) Set(key, value string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.data[key] = value
}

func (v *Values) Delete(key string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.data, key)
	delete(v.resourceByKey, key)
}

func (v *Values) Has(key string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	_, exists := v.data[key]
	return exists
}

func (v *Values) Keys() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	keys := make([]string, 0, len(v.data))
	for key := range v.data {
		keys = append(keys, key)
	}
	return keys
}

func (v *Values) All() map[string]string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	result := make(map[string]string, len(v.data))
	for key, value := range v.data {
		result[key] = value
	}
	return result
}

// ShouldResourceBeUpdated returns whether a resource should be updated
// considering recent use of values it returns
func (v *Values) ShouldResourceBeUpdated(resource string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Clean up old entries first
	now := time.Now()
	for res, lastUpdate := range v.lastUpdateByResource {
		if now.Sub(lastUpdate) >= v.ttl {
			delete(v.lastUpdateByResource, res)
		}
	}

	// Check if resource needs update
	_, exists := v.lastUpdateByResource[resource]
	return !exists
}

// UpdateByResource updates values from a resource and tracks which resource provided them
func (v *Values) UpdateByResource(resource string, data map[string]string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Update the data
	for key, value := range data {
		v.data[key] = value
		v.resourceByKey[key] = resource
	}

	// Mark resource as updated
	v.lastUpdateByResource[resource] = time.Now()
}

func (v *Values) Update(data map[string]string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	for key, value := range data {
		v.data[key] = value
	}
}

func (v *Values) Len() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.data)
}
