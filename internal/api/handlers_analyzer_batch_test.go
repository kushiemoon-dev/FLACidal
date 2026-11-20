package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"

	core "github.com/kushiemoon-dev/flacidal-core"
)

func batchServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	s := NewServer(ServerConfig{Config: &core.Config{DownloadFolder: dir}})
	return s, dir
}

func TestAnalyzeBatch_ConfinementAndProgress(t *testing.T) {
	s, dir := batchServer(t)
	var mu sync.Mutex
	var events []map[string]interface{}
	s.analyzeProgressHook = func(m map[string]interface{}) {
		mu.Lock()
		events = append(events, m)
		mu.Unlock()
	}
	good := filepath.Join(dir, "a.flac")
	os.WriteFile(good, []byte("not audio"), 0o644)
	outside := filepath.Join(t.TempDir(), "b.flac")
	os.WriteFile(outside, []byte("x"), 0o644)

	var body struct {
		Results   []core.AnalysisResult `json:"results"`
		Cancelled bool                  `json:"cancelled"`
		Total     int                   `json:"total"`
	}
	resp := doRequest(t, s, "POST", "/api/analyze/batch", map[string]interface{}{
		"batchId": "b1", "paths": []string{good, outside, good + "2"},
	}, &body)
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if body.Total != 3 || len(body.Results) != 3 || body.Cancelled {
		t.Fatalf("body = %+v", body)
	}
	for _, r := range body.Results {
		if r.Verdict != "error" {
			t.Errorf("garbage and refused files must be error rows: %+v", r)
		}
	}
	var refusedDetails string
	for _, r := range body.Results {
		if r.FilePath == outside {
			refusedDetails = r.Details
		}
	}
	if refusedDetails == "" {
		t.Error("refused path must carry the confinement message")
	}
	if len(events) != 3 {
		t.Fatalf("%d progress events, want 3", len(events))
	}
	for i, e := range events {
		if e["type"] != "analyze-progress" || e["batchId"] != "b1" || e["total"] != 3 || e["done"] != i+1 {
			t.Errorf("event %d = %v", i, e)
		}
	}
}

func TestAnalyzeBatch_Limits(t *testing.T) {
	s, _ := batchServer(t)
	big := make([]string, core.MaxAnalyzeBatch+1)
	for _, route := range []string{"/api/analyze/batch", "/api/analyze/multiple"} {
		var body map[string]interface{}
		resp := doRequest(t, s, "POST", route, map[string]interface{}{"batchId": "x", "paths": big}, &body)
		if resp.StatusCode != fiber.StatusRequestEntityTooLarge {
			t.Errorf("%s status = %d, want 413", route, resp.StatusCode)
		}
	}
	if resp := doRequest(t, s, "POST", "/api/analyze/batch", map[string]interface{}{"batchId": "", "paths": []string{"/a"}}, nil); resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("empty id status = %d", resp.StatusCode)
	}
	if resp := doRequest(t, s, "POST", "/api/analyze/batch", map[string]interface{}{"batchId": "x", "paths": []string{}}, nil); resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("empty paths status = %d", resp.StatusCode)
	}
}

func TestAnalyzeBatch_DuplicateIDAndCancel(t *testing.T) {
	s, _ := batchServer(t)
	ctx, release, err := s.analyzeBatches.Begin("dup")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if resp := doRequest(t, s, "POST", "/api/analyze/batch", map[string]interface{}{"batchId": "dup", "paths": []string{"/a"}}, nil); resp.StatusCode != fiber.StatusConflict {
		t.Errorf("duplicate status = %d, want 409", resp.StatusCode)
	}
	var body map[string]interface{}
	if resp := doRequest(t, s, "POST", "/api/analyze/batch/dup/cancel", nil, &body); resp.StatusCode != fiber.StatusOK || body["cancelled"] != true {
		t.Errorf("cancel: %d %v", resp.StatusCode, body)
	}
	if ctx.Err() == nil {
		t.Error("cancel must cancel the batch context")
	}
	if resp := doRequest(t, s, "POST", "/api/analyze/batch/unknown/cancel", nil, nil); resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("unknown cancel status = %d, want 404", resp.StatusCode)
	}
}

func rawRequest(t *testing.T, s *Server, path string, body interface{}) (int, string, string) {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data), resp.Header.Get("Content-Disposition")
}

func TestAnalyzeReport(t *testing.T) {
	s, _ := batchServer(t)
	results := []core.AnalysisResult{{FilePath: "/m/=x.flac", FileName: "=x.flac", Verdict: "lossless"}}

	status, data, disp := rawRequest(t, s, "/api/analyze/report?format=csv", map[string]interface{}{"results": results})
	if status != 200 || !strings.Contains(disp, `analysis_report.csv`) {
		t.Fatalf("csv: %d %q", status, disp)
	}
	rows, err := csv.NewReader(strings.NewReader(data)).ReadAll()
	if err != nil || len(rows) != 2 || rows[1][1] != "'=x.flac" {
		t.Errorf("csv rows = %v, %v", rows, err)
	}

	status, data, _ = rawRequest(t, s, "/api/analyze/report?format=json", map[string]interface{}{"results": results})
	var back []core.AnalysisResult
	if status != 200 || json.Unmarshal([]byte(data), &back) != nil || len(back) != 1 {
		t.Errorf("json: %d %q", status, data)
	}
	if status, _, _ = rawRequest(t, s, "/api/analyze/report?format=xml", map[string]interface{}{"results": results}); status != 400 {
		t.Errorf("xml status = %d", status)
	}
	if status, _, _ = rawRequest(t, s, "/api/analyze/report?format=csv", map[string]interface{}{"results": []core.AnalysisResult{}}); status != 400 {
		t.Errorf("empty status = %d", status)
	}
	big := make([]core.AnalysisResult, core.MaxAnalyzeBatch+1)
	if status, _, _ = rawRequest(t, s, "/api/analyze/report?format=csv", map[string]interface{}{"results": big}); status != 413 {
		t.Errorf("big status = %d", status)
	}
}
