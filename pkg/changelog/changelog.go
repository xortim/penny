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

var (
	// Matches ## [version] or ## [version] - date
	headerRe = regexp.MustCompile(`^## \[(.+?)\](?:\s*-\s*(.+))?`)
	// Matches ### Heading
	subheadRe = regexp.MustCompile(`(?m)^### (.+)`)
	// Matches [text](url)
	linkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	// Matches **bold**
	boldRe = regexp.MustCompile(`\*\*([^*]+)\*\*`)
)

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

// Latest returns the first (most recent) section formatted as Slack mrkdwn.
func (c Changelog) Latest() (string, error) {
	if len(c.Sections) == 0 {
		return "", fmt.Errorf("changelog is empty")
	}
	return FormatSection(c.Sections[0]), nil
}

// Since returns all sections newer than the specified version, formatted as Slack mrkdwn.
func (c Changelog) Since(version string) (string, error) {
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
		parts = append(parts, FormatSection(s))
	}
	return strings.Join(parts, "\n"), nil
}

// FormatSection converts a changelog section to Slack mrkdwn.
func FormatSection(s Section) string {
	var header string
	if s.Version == "Unreleased" {
		header = "*Latest Changes*"
	} else {
		if s.Date != "" {
			header = fmt.Sprintf("*v%s* (%s)", s.Version, s.Date)
		} else {
			header = fmt.Sprintf("*v%s*", s.Version)
		}
	}

	body := s.Body
	// Convert ### Heading → *Heading*
	body = subheadRe.ReplaceAllString(body, "*$1*")
	// Convert [text](url) → <url|text>
	body = linkRe.ReplaceAllString(body, "<$2|$1>")
	// Convert **bold** → *bold*
	body = boldRe.ReplaceAllString(body, "*$1*")

	return header + "\n" + strings.TrimRight(body, "\n") + "\n"
}
