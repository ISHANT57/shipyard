package main

import "context"

// context_WithRequestID and requestIDFromContext isolate the one
// context.WithValue call this package needs behind typed accessors, so
// every other file just calls a function instead of touching the
// interface{} key/value API directly.
func context_WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
