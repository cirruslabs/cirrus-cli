// Code generated by ogen, DO NOT EDIT.

package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-faster/errors"

	"github.com/ogen-go/ogen/ogenerrors"
)

// SecurityHandler is handler for security parameters.
type SecurityHandler interface {
	// HandleAuthorization handles authorization security.
	HandleAuthorization(ctx context.Context, operationName string, t Authorization) (context.Context, error)
	// HandleCookie handles cookie security.
	HandleCookie(ctx context.Context, operationName string, t Cookie) (context.Context, error)
}

func findAuthorization(h http.Header, prefix string) (string, bool) {
	v, ok := h["Authorization"]
	if !ok {
		return "", false
	}
	for _, vv := range v {
		scheme, value, ok := strings.Cut(vv, " ")
		if !ok || !strings.EqualFold(scheme, prefix) {
			continue
		}
		return value, true
	}
	return "", false
}

func (s *Server) securityAuthorization(ctx context.Context, operationName string, req *http.Request) (context.Context, bool, error) {
	var t Authorization
	token, ok := findAuthorization(req.Header, "Bearer")
	if !ok {
		return ctx, false, nil
	}
	t.Token = token
	rctx, err := s.sec.HandleAuthorization(ctx, operationName, t)
	if errors.Is(err, ogenerrors.ErrSkipServerSecurity) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return rctx, true, err
}
func (s *Server) securityCookie(ctx context.Context, operationName string, req *http.Request) (context.Context, bool, error) {
	var t Cookie
	const parameterName = "_tuist_cloud_key"
	var value string
	switch cookie, err := req.Cookie(parameterName); {
	case err == nil: // if NO error
		value = cookie.Value
	case errors.Is(err, http.ErrNoCookie):
		return ctx, false, nil
	default:
		return nil, false, errors.Wrap(err, "get cookie value")
	}
	t.APIKey = value
	rctx, err := s.sec.HandleCookie(ctx, operationName, t)
	if errors.Is(err, ogenerrors.ErrSkipServerSecurity) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return rctx, true, err
}

// SecuritySource is provider of security values (tokens, passwords, etc.).
type SecuritySource interface {
	// Authorization provides authorization security value.
	Authorization(ctx context.Context, operationName string) (Authorization, error)
	// Cookie provides cookie security value.
	Cookie(ctx context.Context, operationName string) (Cookie, error)
}

func (s *Client) securityAuthorization(ctx context.Context, operationName string, req *http.Request) error {
	t, err := s.sec.Authorization(ctx, operationName)
	if err != nil {
		return errors.Wrap(err, "security source \"Authorization\"")
	}
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return nil
}
func (s *Client) securityCookie(ctx context.Context, operationName string, req *http.Request) error {
	t, err := s.sec.Cookie(ctx, operationName)
	if err != nil {
		return errors.Wrap(err, "security source \"Cookie\"")
	}
	req.AddCookie(&http.Cookie{
		Name:  "_tuist_cloud_key",
		Value: t.APIKey,
	})
	return nil
}