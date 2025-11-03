package engine

import "time"

type CacheValue[T any] struct {
	Value     T
	MaxAge    int64
	expiresAt *time.Time
}

func NewCacheValue[T any](value T, maxAge int64) *CacheValue[T] {
	expiresAt := time.Now().Add(time.Duration(maxAge) * time.Second)
	return &CacheValue[T]{
		Value:     value,
		MaxAge:    maxAge,
		expiresAt: &expiresAt,
	}
}

func (cv *CacheValue[T]) IsExpired() bool {
	if cv.expiresAt == nil {
		return false
	}
	return time.Now().After(*cv.expiresAt)
}
