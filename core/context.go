package core

import "context"

// Context keys for analysis options
type contextKey string

const (
	suppressHeaderKey contextKey = "suppressHeader"
	useFollowKey      contextKey = "useFollow"
)

// withSuppressHeader sets whether headers should be suppressed in the context
func withSuppressHeader(ctx context.Context) context.Context {
	return context.WithValue(ctx, suppressHeaderKey, true)
}

// shouldSuppressHeader returns whether headers should be suppressed from context
func shouldSuppressHeader(ctx context.Context) bool {
	val := ctx.Value(suppressHeaderKey)
	if val == nil {
		return false // default: show headers
	}
	suppress, ok := val.(bool)
	return ok && suppress
}

// withUseFollow sets whether git follow should be used in the context
func withUseFollow(ctx context.Context, useFollow bool) context.Context {
	return context.WithValue(ctx, useFollowKey, useFollow)
}

// shouldUseFollow returns whether git follow should be used from context
func shouldUseFollow(ctx context.Context) bool {
	val := ctx.Value(useFollowKey)
	if val == nil {
		return false // default: don't use follow
	}
	useFollow, ok := val.(bool)
	return ok && useFollow
}
