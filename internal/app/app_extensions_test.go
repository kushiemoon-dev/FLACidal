package app

import (
	"errors"
	"strings"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func TestExtensions_NoCoreReturnsClearError(t *testing.T) {
	a := &App{}
	checks := map[string]error{
		"install":   func() error { _, err := a.InstallExtension("https://x.example/e.zip"); return err }(),
		"registry":  func() error { _, err := a.InstallExtensionFromRegistry("https://github.com/o/r", "x"); return err }(),
		"uninstall": a.UninstallExtension("x"),
		"enable":    a.SetExtensionEnabled("x", true),
		"auth":      a.SetExtensionAuth("x", nil),
		"addReg":    a.AddRegistrySource("https://github.com/o/r"),
		"removeReg": a.RemoveRegistrySource("https://github.com/o/r"),
		"browse":    func() error { _, err := a.BrowseRegistry("https://github.com/o/r"); return err }(),
		"list":      func() error { _, err := a.ListExtensions(); return err }(),
		"sources":   func() error { _, err := a.GetRegistrySources(); return err }(),
		"effective": func() error { _, err := a.GetEffectiveSourceOrder(); return err }(),
	}
	for name, err := range checks {
		if err == nil || !strings.Contains(err.Error(), "core not initialised") {
			t.Errorf("%s: err = %v, want core not initialised", name, err)
		}
	}
}

func TestExtensions_DelegateToCore(t *testing.T) {
	a := newAttachedApp(t, nil)

	list, err := a.ListExtensions()
	if err != nil || len(list) != 0 {
		t.Fatalf("ListExtensions = %v, %v", list, err)
	}
	var ce *core.CodedError
	if _, err := a.InstallExtension("http://example.com/e.zip"); !errors.As(err, &ce) || ce.Code != core.CodeExtensionInsecureURL {
		t.Fatalf("InstallExtension http = %v, want insecure_url", err)
	}
	if err := a.SetExtensionEnabled("missing", true); !errors.As(err, &ce) || ce.Code != core.CodeExtensionNotFound {
		t.Fatalf("SetExtensionEnabled missing = %v", err)
	}
	srcs, err := a.GetRegistrySources()
	if err != nil || len(srcs) != 1 || srcs[0] != core.DefaultRegistrySource {
		t.Fatalf("GetRegistrySources = %v, %v", srcs, err)
	}
	if err := a.AddRegistrySource("https://github.com/someone/exts"); err != nil {
		t.Fatal(err)
	}
	if err := a.RemoveRegistrySource("https://github.com/someone/exts"); err != nil {
		t.Fatal(err)
	}
	order, err := a.GetEffectiveSourceOrder()
	if err != nil || len(order) == 0 || order[len(order)-1] != "soulseek" {
		t.Fatalf("GetEffectiveSourceOrder = %v, %v", order, err)
	}
}
