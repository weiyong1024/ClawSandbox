package container

import (
	"strings"
	"testing"
)

func TestChannelPolicyStepsSlackUsesBooleanAllowBots(t *testing.T) {
	steps := channelPolicySteps("slack", "channels.slack")

	var found bool
	for _, step := range steps {
		if step.path != "channels.slack.allowBots" {
			continue
		}
		found = true
		if step.value != "true" {
			t.Fatalf("expected Slack allowBots value true, got %q", step.value)
		}
		if !step.strictJSON {
			t.Fatal("expected Slack allowBots to use strict JSON")
		}
	}
	if !found {
		t.Fatal("expected Slack allowBots policy step")
	}
}

func TestChannelPolicyStepsDiscordUsesMentionsAllowBots(t *testing.T) {
	steps := channelPolicySteps("discord", "channels.discord")

	var found bool
	for _, step := range steps {
		if step.path != "channels.discord.allowBots" {
			continue
		}
		found = true
		if step.value != "mentions" {
			t.Fatalf("expected Discord allowBots value mentions, got %q", step.value)
		}
		if step.strictJSON {
			t.Fatal("did not expect Discord allowBots to use strict JSON")
		}
	}
	if !found {
		t.Fatal("expected Discord allowBots policy step")
	}
}

func TestChannelPolicyStepsTelegramIncludesGroupAllowFrom(t *testing.T) {
	steps := channelPolicySteps("telegram", "channels.telegram")

	var found bool
	for _, step := range steps {
		if step.path != "channels.telegram.groupAllowFrom" {
			continue
		}
		found = true
		if step.value != `["*"]` {
			t.Fatalf("expected Telegram groupAllowFrom wildcard, got %q", step.value)
		}
		if !step.strictJSON {
			t.Fatal("expected Telegram groupAllowFrom to use strict JSON")
		}
	}
	if !found {
		t.Fatal("expected Telegram groupAllowFrom policy step")
	}
}

func TestRenderSoulMarkdownWithTeammates(t *testing.T) {
	p := SoulParams{
		Name: "Zeus",
		Bio:  "King of the Olympian gods.",
		Teammates: []Teammate{
			{Name: "SunWukong", Bio: "The Monkey King who defied heaven.", Channel: "discord"},
			{Name: "Odin", Bio: "The All-Father of Norse mythology.", Channel: "discord"},
		},
	}
	md := RenderSoulMarkdown(p)

	if !strings.Contains(md, "## Your Team") {
		t.Fatal("expected Your Team section")
	}
	if !strings.Contains(md, "### How to collaborate") {
		t.Fatal("expected How to collaborate subsection")
	}
	if !strings.Contains(md, "**SunWukong**: The Monkey King who defied heaven.") {
		t.Fatal("expected SunWukong teammate entry with full bio")
	}
	if !strings.Contains(md, "(discord)") {
		t.Fatal("expected channel in teammate entry")
	}
	if !strings.Contains(md, "@SunWukong") {
		t.Fatal("expected @mention example using first teammate's name")
	}
	if !strings.Contains(md, "Do NOT @mention a teammate who has already spoken") {
		t.Fatal("expected bounded roundtable termination rule")
	}
	if strings.Contains(md, "**Zeus**") {
		t.Fatal("SOUL.md must not list the character itself as a teammate")
	}
}

func TestRenderSoulMarkdownWithoutTeammates(t *testing.T) {
	p := SoulParams{
		Name: "Zeus",
		Bio:  "King of the Olympian gods.",
	}
	md := RenderSoulMarkdown(p)

	if strings.Contains(md, "## Your Team") {
		t.Fatal("single instance should not have Your Team section")
	}
	if strings.Contains(md, "How to collaborate") {
		t.Fatal("single instance should not have collaboration rules")
	}
	if !strings.Contains(md, "# Zeus") {
		t.Fatal("expected character name heading")
	}
}

func TestRenderSoulMarkdownTeammateFormatting(t *testing.T) {
	tests := []struct {
		name      string
		teammate  Teammate
		wantInMD  string
		wantNoMD  string
	}{
		{
			name:     "no bio",
			teammate: Teammate{Name: "Alice", Channel: "telegram"},
			wantInMD: "- **Alice** (telegram)",
			wantNoMD: ": ",
		},
		{
			name:     "no channel",
			teammate: Teammate{Name: "Bob", Bio: "A data analyst."},
			wantInMD: "- **Bob**: A data analyst.",
			wantNoMD: "()",
		},
		{
			name:     "full bio preserved",
			teammate: Teammate{Name: "Carol", Bio: "A creative writer and storytelling expert with deep knowledge of interactive fiction and narrative design.", Channel: "discord"},
			wantInMD: "interactive fiction and narrative design.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := SoulParams{
				Name:      "Test",
				Teammates: []Teammate{tc.teammate},
			}
			md := RenderSoulMarkdown(p)
			if tc.wantInMD != "" && !strings.Contains(md, tc.wantInMD) {
				t.Fatalf("expected %q in output, got:\n%s", tc.wantInMD, md)
			}
			if tc.wantNoMD != "" && strings.Contains(md, tc.wantNoMD) {
				t.Fatalf("did not expect %q in output, got:\n%s", tc.wantNoMD, md)
			}
		})
	}
}
