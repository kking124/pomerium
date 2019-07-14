package log // import "github.com/pomerium/pomerium/internal/log"

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"testing"

	"github.com/rs/zerolog"
)

func TestGenerateUUID(t *testing.T) {
	prev := uuid()
	for i := 0; i < 100; i++ {
		id := uuid()
		if id == "" {
			t.Fatal("random pool failure")
		}
		if prev == id {
			t.Fatalf("Should get a new ID!")
		}
		matched := regexp.MustCompile("[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}").MatchString(id)
		if !matched {
			t.Fatalf("expected match %s %v", id, matched)
		}
	}
}

func decodeIfBinary(out fmt.Stringer) string {
	return out.String()
}

func TestNewHandler(t *testing.T) {
	log := zerolog.New(nil).With().
		Str("foo", "bar").
		Logger()
	lh := NewHandler(log)
	h := lh(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		if !reflect.DeepEqual(*l, log) {
			t.Fail()
		}
	}))
	h.ServeHTTP(nil, &http.Request{})
}

func TestURLHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		URL: &url.URL{Path: "/path", RawQuery: "foo=bar"},
	}
	h := URLHandler("url")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"url":"/path?foo=bar"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func TestMethodHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		Method: "POST",
	}
	h := MethodHandler("method")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"method":"POST"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func TestRequestHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/path", RawQuery: "foo=bar"},
	}
	h := RequestHandler("request")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"request":"POST /path?foo=bar"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func TestRemoteAddrHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		RemoteAddr: "1.2.3.4:1234",
	}
	h := RemoteAddrHandler("ip")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"ip":"1.2.3.4"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func TestRemoteAddrHandlerIPv6(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		RemoteAddr: "[2001:db8:a0b:12f0::1]:1234",
	}
	h := RemoteAddrHandler("ip")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"ip":"2001:db8:a0b:12f0::1"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func TestUserAgentHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		Header: http.Header{
			"User-Agent": []string{"some user agent string"},
		},
	}
	h := UserAgentHandler("ua")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"ua":"some user agent string"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func TestRefererHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		Header: http.Header{
			"Referer": []string{"http://foo.com/bar"},
		},
	}
	h := RefererHandler("referer")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"referer":"http://foo.com/bar"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func TestRequestIDHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		Header: http.Header{
			"Referer": []string{"http://foo.com/bar"},
		},
	}
	h := RequestIDHandler("id", "Request-Id")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := IDFromRequest(r)
		if !ok {
			t.Fatal("Missing id in request")
		}
		l := FromRequest(r)
		l.Log().Msg("")
		if want, got := fmt.Sprintf(`{"id":"%s"}`+"\n", id), decodeIfBinary(out); want != got {
			t.Errorf("Invalid log output, got: %s, want: %s", got, want)
		}
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(httptest.NewRecorder(), r)
}

func TestCombinedHandlers(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/path", RawQuery: "foo=bar"},
	}
	h := MethodHandler("method")(RequestHandler("request")(URLHandler("url")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"method":"POST","request":"POST /path?foo=bar","url":"/path?foo=bar"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}

func BenchmarkHandlers(b *testing.B) {
	r := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/path", RawQuery: "foo=bar"},
	}
	h1 := URLHandler("url")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h2 := MethodHandler("method")(RequestHandler("request")(h1))
	handlers := map[string]http.Handler{
		"Single":           NewHandler(zerolog.New(ioutil.Discard))(h1),
		"Combined":         NewHandler(zerolog.New(ioutil.Discard))(h2),
		"SingleDisabled":   NewHandler(zerolog.New(ioutil.Discard).Level(zerolog.Disabled))(h1),
		"CombinedDisabled": NewHandler(zerolog.New(ioutil.Discard).Level(zerolog.Disabled))(h2),
	}
	for name := range handlers {
		h := handlers[name]
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				h.ServeHTTP(nil, r)
			}
		})
	}
}

func BenchmarkDataRace(b *testing.B) {
	log := zerolog.New(nil).With().
		Str("foo", "bar").
		Logger()
	lh := NewHandler(log)
	h := lh(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("bar", "baz")
		})
		l.Log().Msg("")
	}))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h.ServeHTTP(nil, &http.Request{})
		}
	})
}

func TestForwardedAddrHandler(t *testing.T) {
	out := &bytes.Buffer{}
	r := &http.Request{
		Header: http.Header{
			"X-Forwarded-For": []string{"client", "proxy1", "proxy2"},
		},
	}
	h := ForwardedAddrHandler("fwd_ip")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := FromRequest(r)
		l.Log().Msg("")
	}))
	h = NewHandler(zerolog.New(out))(h)
	h.ServeHTTP(nil, r)
	if want, got := `{"fwd_ip":"client"}`+"\n", decodeIfBinary(out); want != got {
		t.Errorf("Invalid log output, got: %s, want: %s", got, want)
	}
}
