package domain

import "context"

type userContextKey struct{}

func WithUser(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userContextKey{}, id)
}
func UserID(ctx context.Context) string { id, _ := ctx.Value(userContextKey{}).(string); return id }
