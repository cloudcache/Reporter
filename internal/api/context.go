package api

import (
	"context"
	"net/http"

	"reporter/internal/domain"
)

type contextKey string

const (
	currentUserKey contextKey = "currentUser"
	traceIDKey     contextKey = "traceID"
)

func withCurrentUser(ctx context.Context, user domain.User) context.Context {
	return context.WithValue(ctx, currentUserKey, user)
}

func currentUser(r *http.Request) (domain.User, bool) {
	user, ok := r.Context().Value(currentUserKey).(domain.User)
	return user, ok
}

func actorID(r *http.Request) string {
	user, ok := currentUser(r)
	if !ok {
		return ""
	}
	return user.ID
}

func withTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func traceID(ctx context.Context) string {
	value, _ := ctx.Value(traceIDKey).(string)
	return value
}
