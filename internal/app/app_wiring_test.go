package app

import (
	"slices"
	"sort"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func newAttachedApp(t *testing.T, cfg *core.Config) *App {
	t.Helper()
	dir := t.TempDir()
	core.SetDataDir(dir)
	t.Cleanup(func() { core.SetDataDir("") })
	if cfg != nil {
		if err := core.SaveConfig(cfg); err != nil {
			t.Fatal(err)
		}
	}
	c, err := core.New(dir, core.Options{InitEndpoints: func() {}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Shutdown)
	a := &App{}
	a.attachCore(c)
	return a
}

func registeredSources(a *App) []string {
	var names []string
	for _, info := range a.sourceManager.GetSourcesInfo() {
		names = append(names, info.Name)
	}
	sort.Strings(names)
	return names
}

// The headless server registers exactly this set for the same config
// (cmd/server TestRunRegistersEverySource).
func TestAttachCoreRegistersSameSourcesAsHeadless(t *testing.T) {
	a := newAttachedApp(t, nil)
	want := []string{"amazon", "bandcamp", "deezer", "qobuz", "spotify", "tidal"}
	if got := registeredSources(a); !slices.Equal(got, want) {
		t.Errorf("sources = %v, want %v", got, want)
	}
	if a.soulseekSource == nil {
		t.Error("the standalone sldl handle is missing")
	}
	if _, ok := a.sourceManager.GetSource("soulseek"); ok {
		t.Error("soulseek registered although it is disabled")
	}

	a = newAttachedApp(t, &core.Config{SoulseekEnabled: true})
	if _, ok := a.sourceManager.GetSource("soulseek"); !ok {
		t.Error("native soulseek not registered when enabled")
	}
	if _, isNative := mustSource(t, a, "soulseek").(*core.SoulseekNativeSource); !isNative {
		t.Error("the registered soulseek source is not the native client")
	}
}

func mustSource(t *testing.T, a *App, name string) core.MusicSource {
	t.Helper()
	src, ok := a.sourceManager.GetSource(name)
	if !ok {
		t.Fatalf("source %q not registered", name)
	}
	return src
}

func TestAttachCoreDefaultOrderPutsSoulseekLast(t *testing.T) {
	a := newAttachedApp(t, nil)
	want := []string{"tidal", "qobuz", "amazon", "bandcamp", "soulseek"}
	if got := a.GetConfig().SourceOrder; !slices.Equal(got, want) {
		t.Errorf("default order = %v, want %v", got, want)
	}
}

func TestSelfHostedDefaultOrderThenSoulseekLast(t *testing.T) {
	a := newAttachedApp(t, &core.Config{QobuzPriorityEndpoints: []string{"https://q.example"}})
	want := []string{"qobuz", "tidal", "amazon", "bandcamp", "soulseek"}
	if got := a.GetConfig().SourceOrder; !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestSetSourceOrderThroughCore(t *testing.T) {
	a := newAttachedApp(t, nil)

	if err := a.SetSourceOrder([]string{"soulseek", "tidal", "qobuz"}); err != nil {
		t.Fatal(err)
	}
	if got := a.GetConfig().SourceOrder; got[0] != "soulseek" {
		t.Errorf("soulseek-first order not kept: %v", got)
	}
	if p := a.orchestrator.Priority(); p[0] != "soulseek" {
		t.Errorf("orchestrator not updated: %v", p)
	}

	if err := a.SetSourceOrder([]string{"qobuz", "tidal"}); err != nil {
		t.Fatal(err)
	}
	if got := a.GetConfig().SourceOrder; !slices.Equal(got, []string{"qobuz", "tidal", "soulseek"}) {
		t.Errorf("order = %v, want soulseek appended last", got)
	}
}

func TestSaveConfigReconfiguresLive(t *testing.T) {
	a := newAttachedApp(t, nil)
	cfg := a.GetConfig()
	cfg.AmazonPriorityEndpoints = []string{"https://my-amazon-proxy.example"}
	cfg.SoulseekEnabled = true
	if err := a.SaveConfig(*cfg); err != nil {
		t.Fatal(err)
	}
	snap := a.amazonSource.PoolSnapshot()
	if len(snap) == 0 || snap[0].URL != "https://my-amazon-proxy.example" {
		t.Errorf("priority endpoint not applied: %+v", snap)
	}
	if _, ok := a.sourceManager.GetSource("soulseek"); !ok {
		t.Error("enabling soulseek did not register it")
	}
}
