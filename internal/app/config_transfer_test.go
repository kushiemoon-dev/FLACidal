package app

import (
	"encoding/json"
	"errors"
	"reflect"
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
