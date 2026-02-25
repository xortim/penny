package changelog

import (
	"fmt"
	"regexp"
	"strings"
)

// Section represents a single version entry in the changelog.
type Section struct {
	Version string
	Date    string
	Body    string
}

// Changelog holds parsed changelog sections ordered latest-first.
type Changelog struct {
	Sections []Section
}

// Matches ## [version] or ## [version] - date
var headerRe = regexp.MustCompile(`^## \[(.+?)\](?:\s*-\s*(.+))?`)

// Parse splits raw Keep a Changelog content into sections.
func Parse(raw string) Changelog {
	var sections []Section
	var current *Section

	for _, line := range strings.Split(raw, "\n") {
		if m := headerRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				sections = append(sections, *current)
			}
			current = &Section{
				Version: m[1],
				Date:    strings.TrimSpace(m[2]),
			}
			continue
		}
		if current != nil {
			current.Body += line + "\n"
		}
	}
	if current != nil {
		sections = append(sections, *current)
	}

	return Changelog{Sections: sections}
}

// LatestMarkdown returns the first (most recent) section formatted as standard markdown.
func (c Changelog) LatestMarkdown() (string, error) {
	if len(c.Sections) == 0 {
		return "", fmt.Errorf("changelog is empty")
	}
	return FormatSectionMarkdown(c.Sections[0]), nil
}

// SinceMarkdown returns all sections newer than the specified version, formatted as standard markdown.
func (c Changelog) SinceMarkdown(version string) (string, error) {
	idx := -1
	for i, s := range c.Sections {
		if s.Version == version {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("version %q not found in changelog", version)
	}
	if idx == 0 {
		return "You're up to date!", nil
	}

	var parts []string
	for _, s := range c.Sections[:idx] {
		parts = append(parts, FormatSectionMarkdown(s))
	}
	return strings.Join(parts, "\n"), nil
}

// FormatSectionMarkdown formats a changelog section as standard markdown,
// suitable for Slack's Markdown Block which handles rendering natively.
func FormatSectionMarkdown(s Section) string {
	var header string
	if s.Version == "Unreleased" {
		header = "**Latest Changes**"
	} else {
		if s.Date != "" {
			header = fmt.Sprintf("**v%s** (%s)", s.Version, s.Date)
		} else {
			header = fmt.Sprintf("**v%s**", s.Version)
		}
	}

	return header + "\n" + strings.TrimRight(s.Body, "\n") + "\n"
}
