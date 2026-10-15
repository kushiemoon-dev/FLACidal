package api

import (
	"os"
	"testing"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// NewServer opens the log file under the data dir, so keep every test away
// from the real ~/.flacidal. Tests that need their own data dir still call
// core.SetDataDir themselves.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "flacidal-api-test")
	if err != nil {
		panic(err)
	}
	core.SetDataDir(dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
