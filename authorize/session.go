package authorize

import (
	"errors"
	"net/http"

	"github.com/pomerium/pomerium/config"
	"github.com/pomerium/pomerium/internal/encoding"
	"github.com/pomerium/pomerium/internal/sessions"
	"github.com/pomerium/pomerium/internal/sessions/cookie"
	"github.com/pomerium/pomerium/internal/sessions/header"
	"github.com/pomerium/pomerium/internal/sessions/queryparam"
	"github.com/pomerium/pomerium/internal/urlutil"
)

func loadRawSession(
	req *http.Request,
	cfg *config.Config,
	encoder encoding.MarshalUnmarshaler,
) ([]byte, error) {
	var loaders []sessions.SessionLoader
	cookieStore, err := getCookieStore(cfg, encoder)
	if err != nil {
		return nil, err
	}
	loaders = append(loaders,
		cookieStore,
		header.NewStore(encoder),
		queryparam.NewStore(encoder, urlutil.QuerySession),
	)

	for _, loader := range loaders {
		sess, err := loader.LoadSession(req)
		if err != nil && !errors.Is(err, sessions.ErrNoSessionFound) {
			return nil, err
		} else if err == nil {
			return []byte(sess), nil
		}
	}

	return nil, sessions.ErrNoSessionFound
}

func loadSession(
	encoder encoding.MarshalUnmarshaler,
	rawJWT []byte,
) (*sessions.State, error) {
	var s sessions.State
	err := encoder.Unmarshal(rawJWT, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func getCookieStore(
	cfg *config.Config,
	encoder encoding.MarshalUnmarshaler,
) (sessions.SessionStore, error) {
	cookieStore, err := cookie.NewStore(func() cookie.Options {
		return cookie.Options{
			Name:     cfg.Options.CookieName,
			Domain:   cfg.Options.CookieDomain,
			Secure:   cfg.Options.CookieSecure,
			HTTPOnly: cfg.Options.CookieHTTPOnly,
			Expire:   cfg.Options.CookieExpire,
		}
	}, encoder)
	if err != nil {
		return nil, err
	}
	return cookieStore, nil
}
