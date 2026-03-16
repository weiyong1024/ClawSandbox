package container

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

// Skill represents a single skill available in a container.
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	Eligible    bool   `json:"eligible"`
	Source      string `json:"source"`
	Bundled     bool   `json:"bundled"`
}

// SkillSearchResult represents a search result from ClawHub.
type SkillSearchResult struct {
	Slug  string  `json:"slug"`
	Name  string  `json:"name"`
	Score float64 `json:"score"`
}

// ListSkills returns all skills available in the container by running
// `openclaw skills list --json`.
func ListSkills(cli *docker.Client, containerID string) ([]Skill, error) {
	out, err := dockerExecOutputAs(cli, containerID, "node", []string{
		"openclaw", "skills", "list", "--json",
	})
	if err != nil {
		return nil, fmt.Errorf("skills list: %w", err)
	}

	var result struct {
		Skills []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Emoji       string `json:"emoji"`
			Eligible    bool   `json:"eligible"`
			Source      string `json:"source"`
			Bundled     bool   `json:"bundled"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("parsing skills JSON: %w", err)
	}

	skills := make([]Skill, len(result.Skills))
	for i, s := range result.Skills {
		skills[i] = Skill{
			Name:        s.Name,
			Description: s.Description,
			Emoji:       s.Emoji,
			Eligible:    s.Eligible,
			Source:      s.Source,
			Bundled:     s.Bundled,
		}
	}
	return skills, nil
}

// SearchClawHub searches the ClawHub skill marketplace via `npx clawhub search`.
// Retries up to 3 times on rate limit errors.
func SearchClawHub(cli *docker.Client, containerID, query string) ([]SkillSearchResult, error) {
	var out string
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		out, err = dockerExecOutputAs(cli, containerID, "node", []string{
			"npx", "clawhub", "search", query, "--limit", "10",
		})
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "Rate limit") {
			return nil, fmt.Errorf("clawhub search: %w", err)
		}
		time.Sleep(time.Duration(2*(attempt+1)) * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("clawhub search: %w", err)
	}

	// Parse output lines: "slug  Name  (score)"
	var results []SkillSearchResult
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "✖") {
			continue
		}
		r := parseSearchLine(line)
		if r.Slug != "" {
			results = append(results, r)
		}
	}
	return results, nil
}

// parseSearchLine parses a clawhub search result line like:
// "home-assistant  Home Assistant  (3.738)"
func parseSearchLine(line string) SkillSearchResult {
	// Find the score in parentheses at the end
	scoreStart := strings.LastIndex(line, "(")
	scoreEnd := strings.LastIndex(line, ")")
	var score float64
	if scoreStart > 0 && scoreEnd > scoreStart {
		fmt.Sscanf(line[scoreStart+1:scoreEnd], "%f", &score)
		line = strings.TrimSpace(line[:scoreStart])
	}

	// Split remaining by double-space (slug and name are separated by 2+ spaces)
	parts := strings.SplitN(line, "  ", 2)
	slug := strings.TrimSpace(parts[0])
	name := slug
	if len(parts) > 1 {
		name = strings.TrimSpace(parts[1])
	}

	return SkillSearchResult{Slug: slug, Name: name, Score: score}
}

// InstallSkill installs a community skill from ClawHub into the container.
// Retries up to 3 times on rate limit errors.
func InstallSkill(cli *docker.Client, containerID, slug string) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = dockerExecAs(cli, containerID, "node", []string{
			"npx", "clawhub",
			"--workdir", "/home/node/.openclaw",
			"--dir", "skills",
			"install", slug, "--no-input",
		})
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "Rate limit") {
			return err
		}
		time.Sleep(time.Duration(2*(attempt+1)) * time.Second)
	}
	return err
}

// UninstallSkill removes a community skill from the container.
func UninstallSkill(cli *docker.Client, containerID, slug string) error {
	return dockerExecAs(cli, containerID, "node", []string{
		"npx", "clawhub",
		"--workdir", "/home/node/.openclaw",
		"--dir", "skills",
		"uninstall", slug, "--yes",
	})
}
