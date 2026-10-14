package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"

	"flacidal/internal/app"
)

func healthServer(t *testing.T, upstreamStatus int, mod func(*ServerConfig)) *Server {
	t.Helper()
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if upstreamStatus != http.StatusOK {
			w.WriteHeader(upstreamStatus)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(up.Close)
	svc := core.NewTidalHifiService()
	svc.SetEndpoints([]string{up.URL})
	return newServerWith(t, func(c *ServerConfig) {
		c.QobuzSource = nil // no real network from the default proxy pool
		c.Downloader = svc
		if mod != nil {
			mod(c)
		}
	})
}

func decodeBody[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

func TestHealthRoutes_FreshConfigIsUntested(t *testing.T) {
	s := healthServer(t, http.StatusOK, nil)

	h := decodeBody[[]core.SourceHealth](t, send(t, s, "GET", "/api/sources/health", nil, ""))
	if len(h) != 1 || h[0].Name != "tidal" || h[0].Status != "untested" {
		t.Fatalf("health = %+v, want tidal untested", h)
	}
	for _, ep := range h[0].Endpoints {
		if ep.State != "untested" {
			t.Errorf("endpoint %s = %q, want untested", ep.URL, ep.State)
		}
	}

	eps := decodeBody[[]app.EndpointStatus](t, send(t, s, "GET", "/api/sources/endpoints", nil, ""))
	for _, ep := range eps {
		if ep.Status == "online" {
			t.Errorf("endpoint %s online before any test", ep.URL)
		}
	}
}

func TestHealthRoutes_EmptyServerReturnsArrays(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.QobuzSource = nil })
	for _, path := range []string{"/api/sources/health", "/api/sources/endpoints"} {
		resp := send(t, s, "GET", path, nil, "")
		if raw := decodeBody[json.RawMessage](t, resp); string(raw) != "[]" {
			t.Errorf("%s = %s, want []", path, raw)
		}
	}
}

func TestProbeRoute_SuccessMakesSourceOnline(t *testing.T) {
	s := healthServer(t, http.StatusOK, nil)
	h := decodeBody[[]core.SourceHealth](t, send(t, s, "POST", "/api/sources/health/probe", nil, ""))
	if len(h) != 1 || h[0].Status != "online" || h[0].Endpoints[0].State != "live" {
		t.Fatalf("after probe = %+v, want tidal online with a live endpoint", h)
	}
}

func TestProbeRoute_FailureIsDeadButNotBlacklisted(t *testing.T) {
	s := healthServer(t, http.StatusInternalServerError, nil)
	h := decodeBody[[]core.SourceHealth](t, send(t, s, "POST", "/api/sources/health/probe", nil, ""))
	if len(h) != 1 || h[0].Status != "dead" {
		t.Fatalf("after failed probe = %+v, want tidal dead", h)
	}
	if ep := h[0].Endpoints[0]; ep.State != "untested" || ep.Fails != 0 {
		t.Errorf("endpoint = %+v, a failed probe must not touch routing state", ep)
	}
}

func TestHealthRoutes_RequireToken(t *testing.T) {
	s := healthServer(t, http.StatusOK, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	routes := []struct{ method, path string }{
		{"GET", "/api/sources/health"},
		{"GET", "/api/sources/endpoints"},
		{"POST", "/api/sources/health/probe"},
	}
	for _, r := range routes {
		if got := send(t, s, r.method, r.path, nil, "").StatusCode; got != http.StatusUnauthorized {
			t.Errorf("%s %s without token = %d, want 401", r.method, r.path, got)
		}
		if got := send(t, s, r.method, r.path, map[string]string{"Authorization": "Bearer s3cret"}, "").StatusCode; got != http.StatusOK {
			t.Errorf("%s %s with token = %d, want 200", r.method, r.path, got)
		}
	}
}
