package changelog

import (
	"strings"
	"testing"
)

const testChangelog = `# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Add test coverage, SlackAPI interface, and integration test harness (#34)
- Add threaded reply support and fix early return bug (#37)

### Other

- Auto-join spam channel, Go 1.26, and slack-go v0.18 compat (#36)

## [0.2] - 2026-02-24

### Added

- Add "what's new?" mention command (#14)

## [0.1] - 2026-02-21

### Changed

- Update main (#33)

### Other

- Initial
`

func TestParse(t *testing.T) {
	cl := Parse(testChangelog)

	if len(cl.Sections) != 3 {
		t.Fatalf("Parse() returned %d sections, want 3", len(cl.Sections))
	}

	tests := []struct {
		idx     int
		version string
		date    string
	}{
		{0, "Unreleased", ""},
		{1, "0.2", "2026-02-24"},
		{2, "0.1", "2026-02-21"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			s := cl.Sections[tt.idx]
			if s.Version != tt.version {
				t.Errorf("section[%d].Version = %q, want %q", tt.idx, s.Version, tt.version)
			}
			if s.Date != tt.date {
				t.Errorf("section[%d].Date = %q, want %q", tt.idx, s.Date, tt.date)
			}
			if strings.TrimSpace(s.Body) == "" {
				t.Errorf("section[%d].Body is empty", tt.idx)
			}
		})
	}
}

func TestParseEmpty(t *testing.T) {
	cl := Parse("")
	if len(cl.Sections) != 0 {
		t.Errorf("Parse(\"\") returned %d sections, want 0", len(cl.Sections))
	}
}

func TestLatest(t *testing.T) {
	cl := Parse(testChangelog)
	got, err := cl.Latest()
	if err != nil {
		t.Fatalf("Latest() error: %v", err)
	}
	if !strings.Contains(got, "Latest Changes") {
		t.Errorf("Latest() should contain 'Latest Changes', got:\n%s", got)
	}
	if !strings.Contains(got, "Add test coverage") {
		t.Errorf("Latest() should contain body content, got:\n%s", got)
	}
}

func TestLatestEmpty(t *testing.T) {
	cl := Parse("")
	_, err := cl.Latest()
	if err == nil {
		t.Error("Latest() on empty changelog should return error")
	}
}

func TestSince(t *testing.T) {
	cl := Parse(testChangelog)

	t.Run("since 0.1 returns Latest Changes and 0.2", func(t *testing.T) {
		got, err := cl.Since("0.1")
		if err != nil {
			t.Fatalf("Since(\"0.1\") error: %v", err)
		}
		if !strings.Contains(got, "Latest Changes") {
			t.Errorf("Since(\"0.1\") should contain Latest Changes")
		}
		if !strings.Contains(got, "0.2") {
			t.Errorf("Since(\"0.1\") should contain 0.2")
		}
		if strings.Contains(got, "*v0.1*") {
			t.Errorf("Since(\"0.1\") should not contain 0.1 section itself")
		}
	})

	t.Run("since 0.2 returns only Latest Changes", func(t *testing.T) {
		got, err := cl.Since("0.2")
		if err != nil {
			t.Fatalf("Since(\"0.2\") error: %v", err)
		}
		if !strings.Contains(got, "Latest Changes") {
			t.Errorf("Since(\"0.2\") should contain Latest Changes")
		}
		if strings.Contains(got, "*v0.2*") {
			t.Errorf("Since(\"0.2\") should not contain 0.2 section")
		}
	})

	t.Run("since nonexistent version returns error", func(t *testing.T) {
		_, err := cl.Since("9.9")
		if err == nil {
			t.Error("Since(\"9.9\") should return error for unknown version")
		}
	})

	t.Run("since latest version returns up to date message", func(t *testing.T) {
		got, err := cl.Since("Unreleased")
		if err != nil {
			t.Fatalf("Since(\"Unreleased\") unexpected error: %v", err)
		}
		if got != "You're up to date!" {
			t.Errorf("Since(\"Unreleased\") = %q, want %q", got, "You're up to date!")
		}
	})
}

func TestLatestMarkdown(t *testing.T) {
	cl := Parse(testChangelog)
	got, err := cl.LatestMarkdown()
	if err != nil {
		t.Fatalf("LatestMarkdown() error: %v", err)
	}
	if !strings.Contains(got, "**Latest Changes**") {
		t.Errorf("LatestMarkdown() should contain '**Latest Changes**', got:\n%s", got)
	}
	if !strings.Contains(got, "Add test coverage") {
		t.Errorf("LatestMarkdown() should contain body content, got:\n%s", got)
	}
}

func TestLatestMarkdownEmpty(t *testing.T) {
	cl := Parse("")
	_, err := cl.LatestMarkdown()
	if err == nil {
		t.Error("LatestMarkdown() on empty changelog should return error")
	}
}

func TestSinceMarkdown(t *testing.T) {
	cl := Parse(testChangelog)

	t.Run("since 0.1 returns Latest Changes and 0.2", func(t *testing.T) {
		got, err := cl.SinceMarkdown("0.1")
		if err != nil {
			t.Fatalf("SinceMarkdown(\"0.1\") error: %v", err)
		}
		if !strings.Contains(got, "**Latest Changes**") {
			t.Errorf("SinceMarkdown(\"0.1\") should contain **Latest Changes**")
		}
		if !strings.Contains(got, "**v0.2**") {
			t.Errorf("SinceMarkdown(\"0.1\") should contain **v0.2**")
		}
	})

	t.Run("since 0.2 returns only Latest Changes", func(t *testing.T) {
		got, err := cl.SinceMarkdown("0.2")
		if err != nil {
			t.Fatalf("SinceMarkdown(\"0.2\") error: %v", err)
		}
		if !strings.Contains(got, "**Latest Changes**") {
			t.Errorf("SinceMarkdown(\"0.2\") should contain **Latest Changes**")
		}
		if strings.Contains(got, "**v0.2**") {
			t.Errorf("SinceMarkdown(\"0.2\") should not contain 0.2 section")
		}
	})

	t.Run("since nonexistent version returns error", func(t *testing.T) {
		_, err := cl.SinceMarkdown("9.9")
		if err == nil {
			t.Error("SinceMarkdown(\"9.9\") should return error for unknown version")
		}
	})

	t.Run("since latest version returns up to date message", func(t *testing.T) {
		got, err := cl.SinceMarkdown("Unreleased")
		if err != nil {
			t.Fatalf("SinceMarkdown(\"Unreleased\") unexpected error: %v", err)
		}
		if got != "You're up to date!" {
			t.Errorf("SinceMarkdown(\"Unreleased\") = %q, want %q", got, "You're up to date!")
		}
	})
}

func TestFormatSectionMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		section Section
		want    []string
	}{
		{
			name: "versioned section preserves markdown links and bold",
			section: Section{
				Version: "0.2",
				Date:    "2026-02-24",
				Body:    "### Added\n\n- Add [what's new](https://example.com) command\n- **Bold** feature",
			},
			want: []string{
				"**v0.2** (2026-02-24)",
				"### Added",
				"[what's new](https://example.com)",
				"**Bold**",
			},
		},
		{
			name: "unreleased section",
			section: Section{
				Version: "Unreleased",
				Body:    "### Added\n\n- Something new",
			},
			want: []string{
				"**Latest Changes**",
				"### Added",
				"- Something new",
			},
		},
		{
			name: "versioned section without date",
			section: Section{
				Version: "1.0",
				Body:    "- A change\n",
			},
			want: []string{
				"**v1.0**",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSectionMarkdown(tt.section)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("FormatSectionMarkdown() missing %q in:\n%s", w, got)
				}
			}
		})
	}
}

func TestFormatSection(t *testing.T) {
	tests := []struct {
		name    string
		section Section
		want    []string
	}{
		{
			name: "versioned section with markdown links",
			section: Section{
				Version: "0.2",
				Date:    "2026-02-24",
				Body:    "### Added\n\n- Add [what's new](https://example.com) command\n- **Bold** feature",
			},
			want: []string{
				"*v0.2* (2026-02-24)",
				"*Added*",
				"<https://example.com|what's new>",
				"*Bold*",
			},
		},
		{
			name: "unreleased section",
			section: Section{
				Version: "Unreleased",
				Body:    "### Added\n\n- Something new",
			},
			want: []string{
				"*Latest Changes*",
				"*Added*",
				"- Something new",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSection(tt.section)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("FormatSection() missing %q in:\n%s", w, got)
				}
			}
		})
	}
}
