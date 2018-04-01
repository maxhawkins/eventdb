package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"firebase.google.com/go/auth"
)

// FirebaseProvider is an auth provider backed by Firebase Authentication
type FirebaseProvider struct {
	AuthClient *auth.Client
	AdminUIDs  []string
}

// FromRequest parses an Authorization header or Cookie as a Firebase JWT token.
func (f *FirebaseProvider) FromRequest(r *http.Request) (Info, error) {
	tokenStr, err := parseRequest(r)
	if err != nil {
		return Info{}, err
	}
	if tokenStr == "" {
		return Info{}, nil
	}

	token, err := f.AuthClient.VerifyIDToken(tokenStr)
	if err != nil && strings.Contains(err.Error(), "token has expired") {
		return Info{}, ErrExpired
	} else if err != nil {
		return Info{}, err
	}

	var isAdmin bool
	for _, u := range f.AdminUIDs {
		if u == token.UID {
			isAdmin = true
			break
		}
	}

	return Info{
		ID:      token.UID,
		IsAdmin: isAdmin,
	}, nil
}

func parseRequest(r *http.Request) (string, error) {
	// First try to get it from a cookie
	cookie, err := r.Cookie("jwt")
	if err == nil {
		return cookie.Value, nil
	}

	// Then see if it's in a Bearer token
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", nil
	}

	authParts := strings.Split(auth, " ")
	if len(authParts) != 2 {
		return "", errors.New("malformed Authorization header")
	}

	authType := authParts[0]
	tokenString := authParts[1]

	if authType != "Bearer" {
		return "", fmt.Errorf("unknown auth type %q", authType)
	}

	return tokenString, nil
}
