package backend

import (
	"testing"
)

func TestApplyTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		meta     *FLACMetadata
		original string
		expected string
	}{
		{
			name:     "artist - title",
			template: "{artist} - {title}",
			meta: &FLACMetadata{
				Title:  "One More Time",
				Artist: "Daft Punk",
			},
			original: "/music/old.flac",
			expected: "Daft Punk - One More Time.flac",
		},
		{
			name:     "tracknum - title",
			template: "{tracknumber} - {title}",
			meta: &FLACMetadata{
				Title:       "Digital Love",
				TrackNumber: "05",
			},
			original: "/music/old.flac",
			expected: "05 - Digital Love.flac",
		},
		{
			name:     "tracknum - artist - title",
			template: "{tracknumber} - {artist} - {title}",
			meta: &FLACMetadata{
				Title:       "Harder Better Faster Stronger",
				Artist:      "Daft Punk",
				TrackNumber: "4",
			},
			original: "/music/old.flac",
			expected: "04 - Daft Punk - Harder Better Faster Stronger.flac",
		},
		{
			name:     "artist - album - title",
			template: "{artist} - {album} - {title}",
			meta: &FLACMetadata{
				Title:  "Aerodynamic",
				Artist: "Daft Punk",
				Album:  "Discovery",
			},
			original: "/music/old.flac",
			expected: "Daft Punk - Discovery - Aerodynamic.flac",
		},
		{
			name:     "title only",
			template: "{title}",
			meta: &FLACMetadata{
				Title: "Crescendolls",
			},
			original: "/music/old.flac",
			expected: "Crescendolls.flac",
		},
		{
			name:     "with date",
			template: "{date} - {title}",
			meta: &FLACMetadata{
				Title: "Nightvision",
				Date:  "2001",
			},
			original: "/music/old.flac",
			expected: "2001 - Nightvision.flac",
		},
		{
			name:     "with isrc",
			template: "{isrc} - {title}",
			meta: &FLACMetadata{
				Title: "Superheroes",
				ISRC:  "GBBPF1100012",
			},
			original: "/music/old.flac",
			expected: "GBBPF1100012 - Superheroes.flac",
		},
		{
			name:     "track number with total",
			template: "{tracknumber} - {title}",
			meta: &FLACMetadata{
				Title:       "High Life",
				TrackNumber: "10/14",
			},
			original: "/music/old.flac",
			expected: "10 - High Life.flac",
		},
		{
			name:     "missing fields fallback",
			template: "{artist} - {title}",
			meta: &FLACMetadata{
				Title: "Unknown Track",
			},
			original: "/music/old.flac",
			expected: "Unknown Artist - Unknown Track.flac",
		},
		{
			name:     "single digit track number",
			template: "{tracknumber} - {title}",
			meta: &FLACMetadata{
				Title:       "Veridis Quo",
				TrackNumber: "8",
			},
			original: "/music/old.flac",
			expected: "08 - Veridis Quo.flac",
		},
		{
			name:     "with genre",
			template: "{genre} - {title}",
			meta: &FLACMetadata{
				Title: "Short Circuit",
				Genre: "Electronic",
			},
			original: "/music/old.flac",
			expected: "Electronic - Short Circuit.flac",
		},
		{
			name:     "invalid chars sanitized",
			template: "{artist} - {title}",
			meta: &FLACMetadata{
				Title:  "Track<>Name",
				Artist: "Artist/Test",
			},
			original: "/music/old.flac",
			expected: "ArtistTest - TrackName.flac",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := applyTemplate(tt.template, tt.meta, tt.original)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("applyTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetRenameTemplates(t *testing.T) {
	templates := GetRenameTemplates()

	if len(templates) == 0 {
		t.Error("GetRenameTemplates returned empty list")
	}

	for _, tmpl := range templates {
		if tmpl["name"] == "" {
			t.Error("Template missing 'name' field")
		}
		if tmpl["template"] == "" {
			t.Error("Template missing 'template' field")
		}
	}
}

func TestRenameTemplatesExpected(t *testing.T) {
	templates := GetRenameTemplates()

	expectedTemplates := []map[string]string{
		{"name": "Artist - Title", "template": "{artist} - {title}"},
		{"name": "TrackNum - Title", "template": "{tracknumber} - {title}"},
		{"name": "TrackNum - Artist - Title", "template": "{tracknumber} - {artist} - {title}"},
		{"name": "Artist - Album - Title", "template": "{artist} - {album} - {title}"},
		{"name": "Album - TrackNum - Title", "template": "{album} - {tracknumber} - {title}"},
		{"name": "Title", "template": "{title}"},
	}

	if len(templates) != len(expectedTemplates) {
		t.Errorf("Expected %d templates, got %d", len(expectedTemplates), len(templates))
	}

	for i, expected := range expectedTemplates {
		if templates[i]["name"] != expected["name"] {
			t.Errorf("Template[%d] name = %q, want %q", i, templates[i]["name"], expected["name"])
		}
		if templates[i]["template"] != expected["template"] {
			t.Errorf("Template[%d] template = %q, want %q", i, templates[i]["template"], expected["template"])
		}
	}
}

func TestRenameResultStruct(t *testing.T) {
	result := RenameResult{
		OldPath: "/music/old.flac",
		NewPath: "/music/new.flac",
		Success: true,
	}

	if result.OldPath != "/music/old.flac" {
		t.Errorf("OldPath = %q, want '/music/old.flac'", result.OldPath)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestRenameResultWithError(t *testing.T) {
	result := RenameResult{
		OldPath: "/music/old.flac",
		NewPath: "/music/old.flac",
		Success: false,
		Error:   "Destination file already exists",
	}

	if result.Error != "Destination file already exists" {
		t.Errorf("Error = %q, want 'Destination file already exists'", result.Error)
	}
}

func TestRenamePreviewStruct(t *testing.T) {
	preview := RenamePreview{
		OldPath:  "/music/old.flac",
		OldName:  "old.flac",
		NewName:  "Artist - Title.flac",
		NewPath:  "/music/Artist - Title.flac",
		HasError: false,
	}

	if preview.OldName != "old.flac" {
		t.Errorf("OldName = %q, want 'old.flac'", preview.OldName)
	}
	if preview.HasError {
		t.Error("HasError should be false")
	}
}

func TestRenamePreviewWithError(t *testing.T) {
	preview := RenamePreview{
		OldPath:  "/music/old.flac",
		OldName:  "old.flac",
		NewName:  "old.flac",
		NewPath:  "/music/old.flac",
		HasError: true,
		Error:    "File already exists",
	}

	if !preview.HasError {
		t.Error("HasError should be true")
	}
	if preview.Error != "File already exists" {
		t.Errorf("Error = %q, want 'File already exists'", preview.Error)
	}
}

func TestTemplateVarsStruct(t *testing.T) {
	vars := TemplateVars{
		Title:       "One More Time",
		Artist:      "Daft Punk",
		Album:       "Discovery",
		TrackNumber: "01",
		Date:        "2001",
		Genre:       "Electronic",
		ISRC:        "GBBPF1100001",
		FileName:    "original",
		Ext:         ".flac",
	}

	if vars.Title != "One More Time" {
		t.Errorf("Title = %q, want 'One More Time'", vars.Title)
	}
	if vars.Ext != ".flac" {
		t.Errorf("Ext = %q, want '.flac'", vars.Ext)
	}
}

func BenchmarkApplyTemplate(b *testing.B) {
	meta := &FLACMetadata{
		Title:       "One More Time",
		Artist:      "Daft Punk",
		Album:       "Discovery",
		TrackNumber: "01",
		Date:        "2001",
	}
	template := "{tracknumber} - {artist} - {title}"
	original := "/music/old.flac"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyTemplate(template, meta, original)
	}
}
