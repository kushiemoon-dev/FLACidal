package main

import (
	"fmt"
	"net"
	"strings"
)

type listenSettings struct {
	Addr        string
	Token       string
	Loopback    bool
	CORSOrigins []string
}

// resolveListenSettings reads LISTEN_HOST, PORT, AUTH_TOKEN and CORS_ORIGINS.
// It refuses a non-loopback bind without a token, so the API is never exposed
// unauthenticated by accident.
func resolveListenSettings(getenv func(string) string) (listenSettings, error) {
	host := getenv("LISTEN_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s := listenSettings{
		Addr:     net.JoinHostPort(host, port),
		Token:    strings.TrimSpace(getenv("AUTH_TOKEN")),
		Loopback: host == "localhost" || isLoopbackIP(host),
	}

	if !s.Loopback && s.Token == "" {
		return listenSettings{}, fmt.Errorf("refusing to listen on %s without authentication: set AUTH_TOKEN (or bind LISTEN_HOST=127.0.0.1)", host)
	}

	for _, o := range strings.Split(getenv("CORS_ORIGINS"), ",") {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			return listenSettings{}, fmt.Errorf("CORS_ORIGINS must list explicit origins, '*' is not allowed")
		}
		s.CORSOrigins = append(s.CORSOrigins, o)
	}
	return s, nil
}

func isLoopbackIP(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
