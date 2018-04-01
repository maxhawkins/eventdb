package auth

import "context"

type ctxMarker struct{}

var (
	ctxMarkerKey = &ctxMarker{}
)

// Context decorates a context with an auth.Info object containing information
// about the current user and their privileges.
func Context(ctx context.Context, opts ...ContextOpt) context.Context {
	var info Info

	for _, opt := range opts {
		opt(&info)
	}

	return info.WithContext(ctx)
}

// User returns an auth.Info object stored in a context
func User(ctx context.Context) Info {
	info, ok := ctx.Value(ctxMarkerKey).(Info)
	if !ok {
		return Info{}
	}
	return info
}

// ContextOpt values are passed to auth.Context to construct an auth.Info object
type ContextOpt func(*Info)

// Admin is passed as an argument to Context to set the auth.Info's IsAdmin flag
func Admin(isAdmin bool) ContextOpt {
	return ContextOpt(func(info *Info) {
		info.IsAdmin = isAdmin
	})
}

// ID is passed as an argument to Context to set the auth.Info's ID
func ID(id string) ContextOpt {
	return ContextOpt(func(info *Info) {
		info.ID = id
	})
}
