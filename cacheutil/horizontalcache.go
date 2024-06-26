package cacheutil

import (
	"errors"
	"sync"
)

type HorizontalCache struct {
	horizontalBaseCaches map[string]*HorizontalBaseCache
	mutex                sync.RWMutex
	capacity             uint64
}

func NewHorizontalCache(capacity uint64) *HorizontalCache {
	c := &HorizontalCache{
		horizontalBaseCaches: make(map[string]*HorizontalBaseCache, capacity),
		capacity:             capacity,
	}
	return c
}

func (c *HorizontalCache) Sets(getBaseKey func(value any) string,
	values []any, getKey func(value any) string) error {
	if getBaseKey == nil || len(values) == 0 || getKey == nil {
		return errors.New("getBaseKey, values, or getKey is empty")
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.horizontalBaseCaches == nil {
		return errors.New("horizontalBaseCaches is empty, need call NewHorizontalCache first")
	}

	for _, value := range values {
		baseKey := getBaseKey(value)
		_, ok := c.horizontalBaseCaches[baseKey]
		if !ok {
			c.horizontalBaseCaches[baseKey] = NewHorizontalBaseCache(5)
		}
		key := getKey(value)
		c.horizontalBaseCaches[baseKey].Set(key, value)
	}
	return nil
}

func (c *HorizontalCache) GetsClone(baseKey string) (map[string]any, bool, error) {
	if baseKey == "" {
		return nil, false, errors.New("baseKey is empty")
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.horizontalBaseCaches == nil {
		return nil, false, errors.New("horizontalBaseCaches is empty, need call NewHorizontalCache first")
	}
	horizontalBaseCache, ok := c.horizontalBaseCaches[baseKey]
	if !ok {
		return nil, false, errors.New("not found by baseKey")
	}
	values, err := horizontalBaseCache.GetsClone()
	return values, true, err
}
func (c *HorizontalCache) GetCount(baseKey string) int {
	if baseKey == "" {
		return 0
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.horizontalBaseCaches == nil {
		return 0
	}
	horizontalBaseCache, ok := c.horizontalBaseCaches[baseKey]
	if !ok {
		return 0
	}
	d, err := horizontalBaseCache.GetsUnsafe()
	if err != nil {
		return 0
	}
	if ok {
		return len(d)
	}
	return 0
}

func (c *HorizontalCache) GetCounts() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.horizontalBaseCaches)
}

func (c *HorizontalCache) Reset() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.horizontalBaseCaches != nil {
		for baseKey := range c.horizontalBaseCaches {
			c.horizontalBaseCaches[baseKey].Reset()
		}
	}
	c.horizontalBaseCaches = make(map[string]*HorizontalBaseCache, c.capacity)
}
