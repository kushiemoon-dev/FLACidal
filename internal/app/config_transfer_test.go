package app

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func fullSecrets() core.Config {
	return core.Config{
		DownloadFolder:    "/music",
		TidalClientID:     "tid",
		TidalClientSecret: "tsec",
		SoulseekUsername:  "su",
		SoulseekPassword:  "sp",
		QobuzAppID:        "qid",
		QobuzAppSecret:    "qsec",
		QobuzAuthToken:    "qtok",
	}
}

func TestBuildConfigExportBlanksSevenSecrets(t *testing.T) {
	in := fullSecrets()
	data, err := BuildConfigExport(in)
	if err != nil {
		t.Fatal(err)
	}
	var out configExport
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d", out.SchemaVersion)
	}
	c := out.Config
	for name, v := range map[string]string{
		"TidalClientID": c.TidalClientID, "TidalClientSecret": c.TidalClientSecret,
		"SoulseekUsername": c.SoulseekUsername, "SoulseekPassword": c.SoulseekPassword,
		"QobuzAppID": c.QobuzAppID, "QobuzAppSecret": c.QobuzAppSecret, "QobuzAuthToken": c.QobuzAuthToken,
	} {
		if v != "" {
			t.Errorf("%s not blanked: %q", name, v)
		}
	}
	if c.DownloadFolder != "/music" {
		t.Errorf("DownloadFolder lost: %q", c.DownloadFolder)
	}
	if in.TidalClientSecret != "tsec" {
		t.Error("input config was mutated")
	}
}

func TestMergeImportedConfigKeepsEmptySecretsReplacesNonEmpty(t *testing.T) {
	current := fullSecrets()
	imported := core.Config{DownloadFolder: "/other", QobuzAppID: "new-qid"}
	data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})

	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	if merged.DownloadFolder != "/other" {
		t.Errorf("DownloadFolder = %q", merged.DownloadFolder)
	}
	if merged.QobuzAppID != "new-qid" {
		t.Errorf("non-empty secret not replaced: %q", merged.QobuzAppID)
	}
	if merged.TidalClientSecret != "tsec" || merged.SoulseekPassword != "sp" || merged.QobuzAuthToken != "qtok" {
		t.Errorf("empty secrets not kept: %+v", merged)
	}
	if current.DownloadFolder != "/music" || current.QobuzAppID != "qid" {
		t.Error("current config was mutated")
	}
}

func TestMergeImportedConfigRejectsBadInput(t *testing.T) {
	for name, data := range map[string]string{
		"schema 0": `{"config":{}}`,
		"schema 2": `{"schemaVersion":2,"config":{}}`,
		"corrupt":  `{not json`,
		"empty":    ``,
	} {
		_, err := MergeImportedConfig([]byte(data), core.Config{})
		if !errors.Is(err, ErrNotConfigExport) {
			t.Errorf("%s: err = %v, want ErrNotConfigExport", name, err)
		}
	}
}

func TestExportThenMergeRoundTrip(t *testing.T) {
	current := fullSecrets()
	data, err := BuildConfigExport(current)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged, current) {
		t.Errorf("round trip changed config:\n%+v\n%+v", merged, current)
	}
}

func TestBuildConfigExportBlanksJellyfinKeyAndProxyCredentials(t *testing.T) {
	in := fullSecrets()
	in.JellyfinAPIKey = "jkey"
	in.ProxyURL = "socks5://user:pass@127.0.0.1:1080"
	data, err := BuildConfigExport(in)
	if err != nil {
		t.Fatal(err)
	}
	var out configExport
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Config.JellyfinAPIKey != "" {
		t.Errorf("JellyfinAPIKey not blanked: %q", out.Config.JellyfinAPIKey)
	}
	if out.Config.ProxyURL != "socks5://127.0.0.1:1080" {
		t.Errorf("ProxyURL = %q, want credentials stripped", out.Config.ProxyURL)
	}
}

// Each secret group is only kept when the destinations it can reach are
// unchanged: Qobuz secrets follow the Qobuz endpoints and the proxy, the
// Jellyfin key follows the Jellyfin URL and the proxy. Tidal client and
// Soulseek credentials are never sent to a configurable host, so they stay.
func TestMergeImportedConfigKeepsSecretsOnlyForUnchangedDestinations(t *testing.T) {
	current := fullSecrets()
	current.JellyfinAPIKey = "jkey"

	const evil = "https://evil.example"
	const (
		qobuz    = "qobuz"
		jellyfin = "jellyfin"
	)
	for name, tc := range map[string]struct {
		mutate func(*core.Config)
		drops  []string
	}{
		"QobuzEndpoints":          {func(c *core.Config) { c.QobuzEndpoints = []string{evil} }, []string{qobuz}},
		"QobuzPriorityEndpoints":  {func(c *core.Config) { c.QobuzPriorityEndpoints = []string{evil} }, []string{qobuz}},
		"QobuzProxyEndpoints":     {func(c *core.Config) { c.QobuzProxyEndpoints = []string{evil} }, []string{qobuz}},
		"QobuzCustomEndpoint":     {func(c *core.Config) { c.QobuzCustomEndpoint = evil }, []string{qobuz}},
		"JellyfinURL":             {func(c *core.Config) { c.JellyfinURL = evil }, []string{jellyfin}},
		"ProxyURL":                {func(c *core.Config) { c.ProxyURL = "socks5://evil.example:1080" }, []string{qobuz, jellyfin}},
		"TidalHifiEndpoints":      {func(c *core.Config) { c.TidalHifiEndpoints = []string{evil} }, nil},
		"TidalPriorityEndpoints":  {func(c *core.Config) { c.TidalPriorityEndpoints = []string{evil} }, nil},
		"TidalCustomEndpoint":     {func(c *core.Config) { c.TidalCustomEndpoint = evil }, nil},
		"AmazonPriorityEndpoints": {func(c *core.Config) { c.AmazonPriorityEndpoints = []string{evil} }, nil},
		"AmazonProxyEndpoints":    {func(c *core.Config) { c.AmazonProxyEndpoints = []string{evil} }, nil},
	} {
		t.Run(name, func(t *testing.T) {
			imported := core.Config{}
			tc.mutate(&imported)
			data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})

			merged, err := MergeImportedConfig(data, current)
			if err != nil {
				t.Fatal(err)
			}
			wantDropped := map[string]bool{}
			for _, g := range tc.drops {
				wantDropped[g] = true
			}
			check := func(group, field, got, want string) {
				if wantDropped[group] {
					want = ""
				}
				if got != want {
					t.Errorf("%s = %q, want %q", field, got, want)
				}
			}
			check(qobuz, "QobuzAppID", merged.QobuzAppID, "qid")
			check(qobuz, "QobuzAppSecret", merged.QobuzAppSecret, "qsec")
			check(qobuz, "QobuzAuthToken", merged.QobuzAuthToken, "qtok")
			check(jellyfin, "JellyfinAPIKey", merged.JellyfinAPIKey, "jkey")
			check("", "TidalClientID", merged.TidalClientID, "tid")
			check("", "TidalClientSecret", merged.TidalClientSecret, "tsec")
			check("", "SoulseekUsername", merged.SoulseekUsername, "su")
			check("", "SoulseekPassword", merged.SoulseekPassword, "sp")
		})
	}
}

func TestBuildConfigExportStripsCredentialsFromEveryURLField(t *testing.T) {
	in := fullSecrets()
	in.JellyfinURL = "http://admin:pw@jf.local:8096"
	in.TidalCustomEndpoint = "https://a:b@tidal.home"
	in.QobuzCustomEndpoint = "https://a:b@qobuz.home"
	in.TidalPriorityEndpoints = []string{"https://admin:pw@hifi.home", "https://open.example"}
	in.QobuzProxyEndpoints = []string{"https://a:b@qp.home"}
	in.ProxyURL = "socks5://u:p%zz@h:1080" // unparsable, must not be exported as is

	data, err := BuildConfigExport(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"admin", "pw", "a:b", "u:p"} {
		if strings.Contains(string(data), leak) {
			t.Errorf("export still contains %q: %s", leak, data)
		}
	}
	var out configExport
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if got := out.Config.TidalPriorityEndpoints; !reflect.DeepEqual(got, []string{"https://hifi.home", "https://open.example"}) {
		t.Errorf("TidalPriorityEndpoints = %v", got)
	}
	if in.TidalPriorityEndpoints[0] != "https://admin:pw@hifi.home" {
		t.Error("input slice was mutated")
	}
}

func TestMergeImportedConfigFileProxyCredentialsWin(t *testing.T) {
	current := fullSecrets()
	current.ProxyURL = "socks5://old:oldpw@127.0.0.1:1080"
	imported := core.Config{ProxyURL: "socks5://new:newpw@127.0.0.1:1080"}
	data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})

	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	if merged.ProxyURL != imported.ProxyURL {
		t.Errorf("ProxyURL = %q, want the file's", merged.ProxyURL)
	}
	if merged.QobuzAuthToken != "qtok" {
		t.Errorf("same host, secrets should be kept, got %q", merged.QobuzAuthToken)
	}
}

func TestExportThenMergeKeepsProxyCredentials(t *testing.T) {
	current := fullSecrets()
	current.JellyfinAPIKey = "jkey"
	current.ProxyURL = "socks5://user:pass@127.0.0.1:1080"
	current.JellyfinURL = "http://admin:pw@jf.local:8096"
	current.TidalPriorityEndpoints = []string{"https://admin:pw@hifi.home", "https://open.example"}
	data, err := BuildConfigExport(current)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merged, current) {
		t.Errorf("round trip changed config:\n%+v\n%+v", merged, current)
	}
}

func TestMergeImportedConfigDoesNotRestoreURLCredentialsWhenProxyChanges(t *testing.T) {
	current := fullSecrets()
	current.TidalPriorityEndpoints = []string{"http://admin:pw@192.168.1.10:8000"}
	current.QobuzCustomEndpoint = "http://admin:pw@192.168.1.11:9000"

	imported := core.Config{
		TidalPriorityEndpoints: []string{"http://192.168.1.10:8000"},
		QobuzCustomEndpoint:    "http://192.168.1.11:9000",
		ProxyURL:               "http://evil.example:8080",
	}
	data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})

	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	if got := merged.TidalPriorityEndpoints[0]; got != "http://192.168.1.10:8000" {
		t.Errorf("TidalPriorityEndpoints[0] = %q, credentials must not come back behind a new proxy", got)
	}
	if merged.QobuzCustomEndpoint != "http://192.168.1.11:9000" {
		t.Errorf("QobuzCustomEndpoint = %q, credentials must not come back behind a new proxy", merged.QobuzCustomEndpoint)
	}
}

func TestMergeImportedConfigRestoresListCredentialsByValue(t *testing.T) {
	current := fullSecrets()
	current.TidalPriorityEndpoints = []string{"https://a:1@x.home", "https://b:2@y.home"}

	// The file reorders the list and adds an endpoint.
	imported := core.Config{TidalPriorityEndpoints: []string{"https://y.home", "https://new.example", "https://x.home"}}
	data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})

	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://b:2@y.home", "https://new.example", "https://a:1@x.home"}
	if !reflect.DeepEqual(merged.TidalPriorityEndpoints, want) {
		t.Errorf("TidalPriorityEndpoints = %v, want %v", merged.TidalPriorityEndpoints, want)
	}
}

func TestBuildConfigExportBlanksOpaqueURLsWithCredentials(t *testing.T) {
	for _, raw := range []string{
		"user:pass@host:1080",
		"admin@hifi.home",
		"socks5:u:p@h:1080",
		`https:\\u:p@host`,
		"socks5://admin:1234#xy@proxy.lan:1080",
		"http://admin:/secret@proxy.lan:3128",
		"socks5://u:12/ab@h:1080",
		"https://dGVzdA/tok3n==@jf.example",
	} {
		in := fullSecrets()
		in.ProxyURL = raw
		data, err := BuildConfigExport(in)
		if err != nil {
			t.Fatal(err)
		}
		var out configExport
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatal(err)
		}
		if out.Config.ProxyURL != "" {
			t.Errorf("ProxyURL %q exported as %q, want it blanked", raw, out.Config.ProxyURL)
		}
	}
}

func TestMergeImportedConfigOpaqueFileValueIsADifferentDestination(t *testing.T) {
	current := fullSecrets()
	current.JellyfinAPIKey = "jkey"

	for name, mutate := range map[string]func(*core.Config){
		"proxy":          func(c *core.Config) { c.ProxyURL = "x@evil:8080" },
		"qobuz list":     func(c *core.Config) { c.QobuzEndpoints = []string{"x@evil.example"} },
		"qobuz custom":   func(c *core.Config) { c.QobuzCustomEndpoint = "x@evil.example" },
		"jellyfin url":   func(c *core.Config) { c.JellyfinURL = "x@evil.example" },
		"opaque proxy 2": func(c *core.Config) { c.ProxyURL = "socks5:u:p@evil:1080" },
	} {
		t.Run(name, func(t *testing.T) {
			imported := core.Config{}
			mutate(&imported)
			data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})
			merged, err := MergeImportedConfig(data, current)
			if err != nil {
				t.Fatal(err)
			}
			if merged.QobuzAuthToken != "" && name != "jellyfin url" {
				t.Errorf("QobuzAuthToken = %q, want it dropped", merged.QobuzAuthToken)
			}
			if merged.JellyfinAPIKey != "" && (name == "jellyfin url" || strings.Contains(name, "proxy")) {
				t.Errorf("JellyfinAPIKey = %q, want it dropped", merged.JellyfinAPIKey)
			}
		})
	}
}

func TestExportThenMergeKeepsDuplicateListEntries(t *testing.T) {
	for _, list := range [][]string{
		{"https://a:1@h", "https://b:2@h"},
		{"https://h", "https://a:1@h"},
	} {
		current := fullSecrets()
		current.TidalPriorityEndpoints = list
		data, err := BuildConfigExport(current)
		if err != nil {
			t.Fatal(err)
		}
		merged, err := MergeImportedConfig(data, current)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(merged.TidalPriorityEndpoints, list) {
			t.Errorf("round trip of %v gave %v", list, merged.TidalPriorityEndpoints)
		}
	}
}

func TestMergeImportedConfigRestoresJellyfinCredentialsEvenWithNewProxy(t *testing.T) {
	current := fullSecrets()
	current.JellyfinURL = "http://admin:pw@jf.local:8096"
	imported := core.Config{JellyfinURL: "http://jf.local:8096", ProxyURL: "http://other.example:8080"}
	data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})

	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	if merged.JellyfinURL != current.JellyfinURL {
		t.Errorf("JellyfinURL = %q, want the saved one with its credentials", merged.JellyfinURL)
	}
}

func TestMergeImportedConfigKeepsQobuzSecretsForSameNonEmptyDestinations(t *testing.T) {
	current := fullSecrets()
	current.QobuzEndpoints = []string{"https://qobuz.example"}
	current.QobuzCustomEndpoint = "https://custom.example"
	current.ProxyURL = "socks5://127.0.0.1:1080"

	imported := core.Config{
		QobuzEndpoints:      []string{"https://qobuz.example"},
		QobuzCustomEndpoint: "https://custom.example",
		ProxyURL:            "socks5://127.0.0.1:1080",
	}
	data, _ := json.Marshal(configExport{SchemaVersion: 1, Config: imported})
	merged, err := MergeImportedConfig(data, current)
	if err != nil {
		t.Fatal(err)
	}
	if merged.QobuzAuthToken != "qtok" || merged.QobuzAppID != "qid" || merged.QobuzAppSecret != "qsec" {
		t.Errorf("Qobuz secrets not kept: %+v", merged)
	}
}

// Any config field that looks like a URL must lose its credentials on export.
// Guards against a field missing from urlFields/urlLists, or added to
// core.Config later.
func TestBuildConfigExportStripsEveryURLLikeField(t *testing.T) {
	var cfg core.Config
	v := reflect.ValueOf(&cfg).Elem()
	var set []string
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		lower := strings.ToLower(name)
		if !strings.Contains(lower, "url") && !strings.Contains(lower, "endpoint") && !strings.Contains(lower, "proxy") {
			continue
		}
		switch f := v.Field(i); {
		case f.Kind() == reflect.String:
			f.SetString("https://u:p@h")
			set = append(set, name)
		case f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.String:
			f.Set(reflect.ValueOf([]string{"https://u:p@h"}))
			set = append(set, name)
		}
	}
	if len(set) < 11 {
		t.Fatalf("only %d URL-like fields found (%v), the guard is not looking at the right fields", len(set), set)
	}
	data, err := BuildConfigExport(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "u:p") {
		t.Errorf("export still holds credentials; fields set: %v\n%s", set, data)
	}
}
