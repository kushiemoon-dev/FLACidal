package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())
	return port
}

// isolate points Core at throwaway directories so run never touches ~/.flacidal.
func isolate(t *testing.T) {
	t.Helper()
	core.SetDataDir(t.TempDir())
	t.Cleanup(func() { core.SetDataDir("") })
	t.Setenv("HOME", t.TempDir())
	t.Setenv("DOWNLOAD_FOLDER", t.TempDir())
}

type runHandle struct {
	base   string
	cancel context.CancelFunc
	result chan error
}

// stop cancels run and returns its result, failing if it does not return.
func (h runHandle) stop(t *testing.T) error {
	t.Helper()
	h.cancel()
	select {
	case err := <-h.result:
		h.result <- err // keep it readable for the cleanup
		return err
	case <-time.After(shutdownTimeout + 10*time.Second):
		t.Fatal("run did not return after cancel")
		return nil
	}
}

func startRun(t *testing.T, env map[string]string) runHandle {
	t.Helper()
	return startRunWith(t, env, nil)
}

// startRunWith runs setup after the isolation, before run starts, to seed Core state.
func startRunWith(t *testing.T, env map[string]string, setup func()) runHandle {
	t.Helper()
	isolate(t)
	if setup != nil {
		setup()
	}
	port := freePort(t)
	full := map[string]string{"PORT": port, "LISTEN_HOST": "127.0.0.1"}
	for k, v := range env {
		full[k] = v
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 2)
	go func() {
		result <- run(ctx, envOf(full), runDeps{initEndpoints: func() {}})
	}()
	h := runHandle{base: "http://127.0.0.1:" + port, cancel: cancel, result: result}
	t.Cleanup(func() {
		cancel()
		select {
		case <-result:
		case <-time.After(10 * time.Second):
		}
	})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-result:
			result <- err // keep it readable for the cleanup
			t.Fatalf("run returned early: %v", err)
		default:
		}
		resp, err := http.Get(h.base + "/api/health")
		if err == nil {
			resp.Body.Close()
			return h
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("server did not become healthy in 5s")
	return h
}

func get(t *testing.T, url, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestRunStartsAndStopsCleanly(t *testing.T) {
	h := startRun(t, nil)
	if code, _ := get(t, h.base+"/api/health", ""); code != http.StatusOK {
		t.Errorf("/api/health = %d, want 200", code)
	}
	if err := h.stop(t); err != nil {
		t.Errorf("run() = %v, want nil after a clean shutdown", err)
	}
}

func TestRunAuthToken(t *testing.T) {
	h := startRun(t, map[string]string{"AUTH_TOKEN": "secret-token"})
	defer h.stop(t) //nolint:errcheck

	if code, _ := get(t, h.base+"/api/health", ""); code != http.StatusOK {
		t.Errorf("/api/health without token = %d, want 200", code)
	}
	if code, _ := get(t, h.base+"/api/version", ""); code != http.StatusUnauthorized {
		t.Errorf("/api/version without token = %d, want 401", code)
	}
	if code, _ := get(t, h.base+"/api/version", "secret-token"); code != http.StatusOK {
		t.Errorf("/api/version with token = %d, want 200", code)
	}
}

func TestRunServesFrontendDir(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html>run-test-index</html>"), 0644); err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	h := startRun(t, map[string]string{"FRONTEND_DIST_DIR": dist})
	defer h.stop(t) //nolint:errcheck

	code, body := get(t, h.base+"/", "")
	if code != http.StatusOK || !strings.Contains(body, "run-test-index") {
		t.Errorf("GET / = %d %q, want the index.html from FRONTEND_DIST_DIR", code, body)
	}
}

func TestRunRejectsInvalidSettings(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"non-loopback without token", map[string]string{"LISTEN_HOST": "0.0.0.0"}, "AUTH_TOKEN"},
		{"wildcard CORS", map[string]string{"CORS_ORIGINS": "*"}, "CORS_ORIGINS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolate(t)
			env := map[string]string{"PORT": freePort(t), "LISTEN_HOST": "127.0.0.1"}
			for k, v := range tt.env {
				env[k] = v
			}
			err := run(context.Background(), envOf(env), runDeps{initEndpoints: func() {}})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("run() = %v, want an error mentioning %s", err, tt.wantErr)
			}
		})
	}
}

func TestRunFailsWhenPortIsTaken(t *testing.T) {
	isolate(t)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())

	done := make(chan error, 1)
	go func() {
		done <- run(context.Background(), envOf(map[string]string{"PORT": port, "LISTEN_HOST": "127.0.0.1"}), runDeps{initEndpoints: func() {}})
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("run() = nil, want an error when the port is taken")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run blocked on a taken port")
	}
}

func TestRunWithSavedConfig(t *testing.T) {
	h := startRunWith(t, nil, func() {
		cfg := core.GetDefaultConfig()
		cfg.ConcurrentDownloads = 0
		cfg.QobuzAuthToken = "qobuz-token"
		cfg.QobuzPriorityEndpoints = []string{"https://qobuz.self-hosted.example", "ftp://rejected.example"}
		cfg.TidalPriorityEndpoints = []string{"https://tidal.self-hosted.example"}
		cfg.SoulseekEnabled = true
		cfg.SoulseekUsername = "user"
		cfg.SoulseekPassword = "pass"
		cfg.SoulseekBinaryPath = filepath.Join(t.TempDir(), "missing-sldl")
		if err := core.SaveConfig(cfg); err != nil {
			t.Fatalf("test setup failed: %v", err)
		}
	})

	code, body := get(t, h.base+"/api/config", "")
	if code != http.StatusOK {
		t.Fatalf("/api/config = %d, want 200", code)
	}
	for _, want := range []string{`"soulseekEnabled":true`, "qobuz.self-hosted.example"} {
		if !strings.Contains(body, want) {
			t.Errorf("/api/config does not reflect the saved config, missing %q in %s", want, body)
		}
	}
	if err := h.stop(t); err != nil {
		t.Errorf("run() = %v, want nil", err)
	}
}

// An idle keep-alive style connection must not hold the shutdown hostage.
func TestRunShutsDownWithIdleConnection(t *testing.T) {
	h := startRun(t, nil)
	conn, err := net.Dial("tcp", strings.TrimPrefix(h.base, "http://"))
	if err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	defer conn.Close()

	start := time.Now()
	if err := h.stop(t); err != nil {
		t.Errorf("run() = %v, want nil", err)
	}
	if d := time.Since(start); d > shutdownTimeout+5*time.Second {
		t.Errorf("shutdown took %s with an idle connection", d)
	}
}

// A signal that lands during startup must still end run.
func TestRunWithAlreadyCancelledContext(t *testing.T) {
	isolate(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	port := freePort(t)
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, envOf(map[string]string{"PORT": port, "LISTEN_HOST": "127.0.0.1"}), runDeps{initEndpoints: func() {}})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("run() = %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return with a cancelled context")
	}
}
