package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// localOnly guards a loopback server that has no token. The Host check blocks
// DNS rebinding; the Origin check blocks a page on another site from posting
// to 127.0.0.1 (a form POST needs no CORS preflight) or opening the WebSocket.
func localOnly(allowedOrigins []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		host := c.Hostname()
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.Trim(strings.ToLower(host), "[]")
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "host not allowed"})
		}
		if origin := c.Get("Origin"); origin != "" && !originAllowed(c, origin, allowedOrigins) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "origin not allowed"})
		}
		return c.Next()
	}
}

// originAllowed accepts the server's own origin (what the bundled SPA sends)
// and the explicitly configured ones.
func originAllowed(c *fiber.Ctx, origin string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, origin) {
			return true
		}
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host != "" && strings.EqualFold(u.Host, c.Get("Host"))
}

func isProtectedPath(path string) bool {
	// Fiber routes case-insensitively, so /API/config reaches /api/config.
	path = strings.ToLower(path)
	if path == "/api/health" {
		return false
	}
	return path == "/api" || strings.HasPrefix(path, "/api/") ||
		path == "/ws" || strings.HasPrefix(path, "/ws/")
}

// tokenAuth checks a Bearer token on /api and /ws. The query token is
// accepted for WebSocket upgrades only (browsers cannot set headers there),
// never for plain HTTP routes where it would end up in access logs.
func tokenAuth(token string) fiber.Handler {
	want := sha256.Sum256([]byte(token))
	return func(c *fiber.Ctx) error {
		if !isProtectedPath(c.Path()) {
			return c.Next()
		}
		got := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
		if !strings.HasPrefix(c.Get("Authorization"), "Bearer ") {
			got = ""
		}
		if got == "" && websocket.IsWebSocketUpgrade(c) {
			got = c.Query("token")
		}
		have := sha256.Sum256([]byte(got))
		if subtle.ConstantTimeCompare(have[:], want[:]) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		return c.Next()
	}
}
