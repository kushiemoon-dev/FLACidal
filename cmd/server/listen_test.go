package main

import (
	"strings"
	"testing"
)

func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveListenSettings(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantErr     bool
		addr        string
		loopback    bool
		token       string
		corsOrigins int
		errMustHave string
		errMustLack string
	}{
		{name: "defaults", env: nil, addr: "127.0.0.1:8080", loopback: true},
		{name: "any interface without token", env: map[string]string{"LISTEN_HOST": "0.0.0.0"}, wantErr: true, errMustHave: "AUTH_TOKEN"},
		{name: "any interface with token", env: map[string]string{"LISTEN_HOST": "0.0.0.0", "AUTH_TOKEN": "tok"}, addr: "0.0.0.0:8080", token: "tok"},
		{name: "LAN address without token", env: map[string]string{"LISTEN_HOST": "192.168.1.10"}, wantErr: true, errMustHave: "AUTH_TOKEN"},
		{name: "localhost", env: map[string]string{"LISTEN_HOST": "localhost", "PORT": "9000"}, addr: "localhost:9000", loopback: true},
		{name: "ipv6 loopback", env: map[string]string{"LISTEN_HOST": "::1"}, addr: "[::1]:8080", loopback: true},
		{name: "loopback with token", env: map[string]string{"AUTH_TOKEN": "tok"}, addr: "127.0.0.1:8080", loopback: true, token: "tok"},
		{name: "blank token is empty", env: map[string]string{"AUTH_TOKEN": "   "}, addr: "127.0.0.1:8080", loopback: true},
		{name: "blank token on non-loopback", env: map[string]string{"LISTEN_HOST": "0.0.0.0", "AUTH_TOKEN": "  "}, wantErr: true},
		{name: "cors list", env: map[string]string{"CORS_ORIGINS": "https://a, https://b,,"}, addr: "127.0.0.1:8080", loopback: true, corsOrigins: 2},
		{name: "cors wildcard", env: map[string]string{"CORS_ORIGINS": "https://a,*"}, wantErr: true},
		{name: "error never echoes token", env: map[string]string{"LISTEN_HOST": "0.0.0.0", "CORS_ORIGINS": "*", "AUTH_TOKEN": "topsecretvalue"}, wantErr: true, errMustLack: "topsecretvalue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveListenSettings(envOf(tt.env))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				if tt.errMustHave != "" && !strings.Contains(err.Error(), tt.errMustHave) {
					t.Errorf("error %q lacks %q", err, tt.errMustHave)
				}
				if tt.errMustLack != "" && strings.Contains(err.Error(), tt.errMustLack) {
					t.Errorf("error %q leaks %q", err, tt.errMustLack)
				}
				return
			}
			if got.Addr != tt.addr || got.Loopback != tt.loopback || got.Token != tt.token || len(got.CORSOrigins) != tt.corsOrigins {
				t.Errorf("got %+v", got)
			}
		})
	}
}
