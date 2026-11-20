package api

import (
	"net"
	"net/url"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	core "github.com/kushiemoon-dev/flacidal-core"

	"flacidal/internal/app"
)

// listenWS serves s on a loopback port and returns an open /ws client.
func listenWS(t *testing.T, s *Server) *websocket.Conn {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.app.Listener(ln) }()
	u := url.URL{Scheme: "ws", Host: ln.Addr().String(), Path: "/ws"}
	var conn *websocket.Conn
	for i := 0; i < 50; i++ {
		if conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	// Cleanups run last in first out: stop the server before the client hangs up.
	t.Cleanup(func() { conn.Close() })
	t.Cleanup(func() { _ = s.Shutdown() })
	time.Sleep(50 * time.Millisecond) // let the hub register the client
	return conn
}

func readWS(t *testing.T, conn *websocket.Conn) map[string]interface{} {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var m map[string]interface{}
	if err := conn.ReadJSON(&m); err != nil {
		t.Fatalf("read ws: %v", err)
	}
	return m
}

func TestParityRoutesRequireToken(t *testing.T) {
	s := newServerWith(t, func(c *ServerConfig) { c.AuthToken = "s3cret" })
	for _, p := range []string{"/api/discography/expand", "/api/discography/queue", "/api/artists/assets", "/api/ffmpeg/install"} {
		if resp := send(t, s, "POST", p, nil, "{}"); resp.StatusCode != 401 {
			t.Errorf("%s: status %d, want 401", p, resp.StatusCode)
		}
	}
}

func TestParityOutputDirIsConfined(t *testing.T) {
	f := newGuardFixture(t)
	outside := filepath.Dir(f.outside)
	routes := map[string]interface{}{
		"/api/discography/queue": map[string]interface{}{"albumUrls": []string{}, "outputDir": outside},
		"/api/artists/assets":    map[string]interface{}{"artistId": "1", "outputDir": outside},
	}
	for path, body := range routes {
		if resp := doRequest(t, f.s, "POST", path, body, nil); resp.StatusCode != 403 {
			t.Errorf("%s: status %d, want 403", path, resp.StatusCode)
		}
	}
}

func TestParityBadJSONIs400(t *testing.T) {
	s := newTestServer(t)
	for _, p := range []string{"/api/discography/expand", "/api/discography/queue", "/api/artists/assets"} {
		if resp := send(t, s, "POST", p, nil, "{not json"); resp.StatusCode != 400 {
			t.Errorf("%s: status %d, want 400", p, resp.StatusCode)
		}
	}
}

func TestParityExpandRejectsNonSpotifyURL(t *testing.T) {
	s := newTestServer(t)
	var out map[string]string
	resp := doRequest(t, s, "POST", "/api/discography/expand", map[string]string{"url": "https://example.com/x"}, &out)
	if resp.StatusCode != 400 || out["error"] == "" {
		t.Errorf("status %d body %v", resp.StatusCode, out)
	}
}

func TestParityWithoutCoreIs503(t *testing.T) {
	f := newGuardFixture(t)
	cases := []struct {
		path string
		body interface{}
	}{
		{"/api/discography/expand", map[string]string{"url": "https://open.spotify.com/artist/abc/discography/all"}},
		{"/api/discography/queue", map[string]interface{}{"albumUrls": []string{}, "outputDir": f.root}},
		{"/api/artists/assets", map[string]interface{}{"artistId": "1", "outputDir": f.root}},
	}
	for _, c := range cases {
		if resp := doRequest(t, f.s, "POST", c.path, c.body, nil); resp.StatusCode != 503 {
			t.Errorf("%s: status %d, want 503", c.path, resp.StatusCode)
		}
	}
}

func TestFFmpegInstallBroadcastsProgressAndRefusesSecondRun(t *testing.T) {
	var mu sync.Mutex
	resets := 0
	started := make(chan struct{})
	release := make(chan struct{})
	restore := app.SetFFmpegInstallForTest(func(ch chan<- core.FFmpegInstallProgress) error {
		defer close(ch)
		ch <- core.FFmpegInstallProgress{Stage: "downloading", Percent: 40}
		close(started)
		<-release
		ch <- core.FFmpegInstallProgress{Stage: "done", Percent: 100}
		return nil
	}, func() { mu.Lock(); resets++; mu.Unlock() })
	defer restore()

	s := newTestServer(t)
	conn := listenWS(t, s)

	done := make(chan int, 1)
	go func() { done <- doRequest(t, s, "POST", "/api/ffmpeg/install", nil, nil).StatusCode }()
	<-started

	if resp := doRequest(t, s, "POST", "/api/ffmpeg/install", nil, nil); resp.StatusCode != 409 {
		t.Errorf("second install: status %d, want 409", resp.StatusCode)
	}
	close(release)
	if code := <-done; code != 200 {
		t.Fatalf("first install: status %d", code)
	}

	first, second := readWS(t, conn), readWS(t, conn)
	if first["type"] != "ffmpeg-install-progress" || first["stage"] != "downloading" || first["percent"] != 40.0 {
		t.Errorf("first event = %v", first)
	}
	if second["stage"] != "done" {
		t.Errorf("second event = %v", second)
	}
	mu.Lock()
	defer mu.Unlock()
	if resets != 1 {
		t.Errorf("converter reset %d times, want 1", resets)
	}
}

func TestLogEntriesAreBroadcastOnceOverWS(t *testing.T) {
	s := newTestServer(t)
	conn := listenWS(t, s)

	s.logBuffer.Info("hello ws")
	m := readWS(t, conn)
	if m["type"] != "log" || m["level"] != "info" || m["message"] != "hello ws" || m["timestamp"] == "" {
		t.Errorf("event = %v", m)
	}
	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	var extra map[string]interface{}
	if err := conn.ReadJSON(&extra); err == nil {
		t.Errorf("duplicate event: %v", extra)
	}
}
