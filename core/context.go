package core

import (
	"context"

	"github.com/huangsam/hotspot/internal/contract"
)

// Context keys for analysis options.
type contextKey string

const (
	suppressHeaderKey contextKey = "suppressHeader"
	useFollowKey      contextKey = "useFollow"
	analysisIDKey     contextKey = "analysisID"
)

// WithSuppressHeader sets whether headers should be suppressed in the context.
func WithSuppressHeader(ctx context.Context) context.Context {
	return context.WithValue(ctx, suppressHeaderKey, true)
}

// shouldSuppressHeader returns whether headers should be suppressed from context.
func shouldSuppressHeader(ctx context.Context) bool {
	val := ctx.Value(suppressHeaderKey)
	if val == nil {
		return false // default: show headers
	}
	suppress, ok := val.(bool)
	return ok && suppress
}

// withUseFollow sets whether git follow should be used in the context.
func withUseFollow(ctx context.Context, useFollow bool) context.Context {
	return context.WithValue(ctx, useFollowKey, useFollow)
}

// shouldUseFollow returns whether git follow should be used from context.
func shouldUseFollow(ctx context.Context) bool {
	val := ctx.Value(useFollowKey)
	if val == nil {
		return false // default: don't use follow
	}
	useFollow, ok := val.(bool)
	return ok && useFollow
}

// withAnalysisID sets the analysis ID in the context.
func withAnalysisID(ctx context.Context, analysisID int64) context.Context {
	return context.WithValue(ctx, analysisIDKey, analysisID)
}

// getAnalysisID returns the analysis ID from context.
func getAnalysisID(ctx context.Context) (int64, bool) {
	val := ctx.Value(analysisIDKey)
	if val == nil {
		return 0, false
	}
	id, ok := val.(int64)
	return id, ok
}

// cacheManagerKey is the context key for the cache manager.
type cacheManagerKeyType struct{}

// contextWithCacheManager returns a new context with the given CacheManager.
func contextWithCacheManager(ctx context.Context, mgr contract.CacheManager) context.Context {
	return context.WithValue(ctx, cacheManagerKeyType{}, mgr)
}

// cacheManagerFromContext retrieves the CacheManager from the context.
func cacheManagerFromContext(ctx context.Context) contract.CacheManager {
	val := ctx.Value(cacheManagerKeyType{})
	if mgr, ok := val.(contract.CacheManager); ok {
		return mgr
	}
	return nil
}
