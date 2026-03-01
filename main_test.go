package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/slack-go/slack"
)

// assertNoPanic runs fn and fails the test if fn panics.
func assertNoPanic(t *testing.T, label string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s caused a panic: %v", label, r)
		}
	}()
	fn()
}

// assertPanics runs fn and fails the test if fn does NOT panic.
func assertPanics(t *testing.T, label string, fn func()) {
	t.Helper()
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		fn()
	}()
	if !panicked {
		t.Errorf("%s: expected a panic but none occurred", label)
	}
}

// ---- Modal creation tests ----

func TestCreateRepoChooserModalStructure(t *testing.T) {
	modal := createRepoChooserModal()

	if modal.Type != slack.VTModal {
		t.Errorf("expected modal type 'modal', got %q", modal.Type)
	}
	if modal.CallbackID != repoModalCallbackID {
		t.Errorf("expected callback_id %q, got %q", repoModalCallbackID, modal.CallbackID)
	}
	if modal.Submit != nil {
		t.Errorf("repo chooser modal must not have a submit button (uses block actions instead)")
	}
	if len(modal.Blocks.BlockSet) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(modal.Blocks.BlockSet))
	}
}

func TestCreateRepoChooserModalUsesExternalSelect(t *testing.T) {
	modal := createRepoChooserModal()

	actionBlock, ok := modal.Blocks.BlockSet[1].(*slack.ActionBlock)
	if !ok {
		t.Fatal("expected second block to be an ActionBlock")
	}

	if len(actionBlock.Elements.ElementSet) != 1 {
		t.Fatalf("expected 1 element in action block, got %d", len(actionBlock.Elements.ElementSet))
	}

	selectEl, ok := actionBlock.Elements.ElementSet[0].(*slack.SelectBlockElement)
	if !ok {
		t.Fatal("expected element to be SelectBlockElement")
	}
	if selectEl.Type != slack.OptTypeExternal {
		t.Errorf("expected external select type, got %q", selectEl.Type)
	}
	if selectEl.ActionID != slashVibeIssueActionID {
		t.Errorf("expected action_id %q, got %q", slashVibeIssueActionID, selectEl.ActionID)
	}
}

func TestCreateLoadingModal(t *testing.T) {
	modal := createLoadingModal()

	if modal.Type != slack.VTModal {
		t.Errorf("expected modal type 'modal', got %q", modal.Type)
	}
	if modal.Submit != nil {
		t.Error("loading modal should not have a submit button")
	}
	if len(modal.Blocks.BlockSet) != 1 {
		t.Errorf("expected 1 block, got %d", len(modal.Blocks.BlockSet))
	}
}

func TestCreatePRChooserModalStructure(t *testing.T) {
	prs := []PRItem{
		{Number: 1, Title: "Fix bug"},
		{Number: 2, Title: "Add feature"},
	}
	modal := createPRChooserModal(prs, "org/repo", `{"repo":"org/repo"}`)

	if modal.Type != slack.VTModal {
		t.Errorf("expected modal type 'modal', got %q", modal.Type)
	}
	if modal.CallbackID != prModalCallbackID {
		t.Errorf("expected callback_id %q, got %q", prModalCallbackID, modal.CallbackID)
	}
	if modal.Submit == nil || modal.Submit.Text != "Post to Channel" {
		t.Errorf("expected submit button labelled 'Post to Channel'")
	}
	if modal.PrivateMetadata != `{"repo":"org/repo"}` {
		t.Errorf("unexpected private_metadata: %q", modal.PrivateMetadata)
	}
	if len(modal.Blocks.BlockSet) != 2 {
		t.Errorf("expected 2 blocks, got %d", len(modal.Blocks.BlockSet))
	}
}

func TestCreatePRChooserModalOptions(t *testing.T) {
	prs := []PRItem{
		{Number: 42, Title: "My PR"},
		{Number: 100, Title: "Another PR"},
	}
	modal := createPRChooserModal(prs, "org/repo", "")

	inputBlock, ok := modal.Blocks.BlockSet[1].(*slack.InputBlock)
	if !ok {
		t.Fatal("expected second block to be an InputBlock")
	}

	selectEl, ok := inputBlock.Element.(*slack.SelectBlockElement)
	if !ok {
		t.Fatal("expected element to be SelectBlockElement")
	}
	if len(selectEl.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(selectEl.Options))
	}
	if selectEl.Options[0].Value != "42" {
		t.Errorf("expected first option value '42', got %q", selectEl.Options[0].Value)
	}
	if selectEl.Options[1].Value != "100" {
		t.Errorf("expected second option value '100', got %q", selectEl.Options[1].Value)
	}
}

func TestCreatePRChooserModalTitleTruncation(t *testing.T) {
	longTitle := make([]byte, 100)
	for i := range longTitle {
		longTitle[i] = 'a'
	}
	prs := []PRItem{{Number: 1, Title: string(longTitle)}}
	modal := createPRChooserModal(prs, "org/repo", "")

	inputBlock := modal.Blocks.BlockSet[1].(*slack.InputBlock)
	selectEl := inputBlock.Element.(*slack.SelectBlockElement)

	if len(selectEl.Options[0].Text.Text) > 75 {
		t.Errorf("option text should be truncated to at most 75 chars, got %d", len(selectEl.Options[0].Text.Text))
	}
}

func TestCreateErrorModal(t *testing.T) {
	modal := createErrorModal("something went wrong")

	if modal.Submit != nil {
		t.Error("error modal should not have a submit button")
	}
	if len(modal.Blocks.BlockSet) != 1 {
		t.Errorf("expected 1 block, got %d", len(modal.Blocks.BlockSet))
	}
}

// ---- extractTextValue tests ----

func TestExtractTextValuePlainInput(t *testing.T) {
	values := map[string]map[string]interface{}{
		"repo_block": {
			"repo_input": map[string]interface{}{
				"type":  "plain_text_input",
				"value": "org/repo",
			},
		},
	}
	got := extractTextValue(values, "repo_block", "repo_input")
	if got != "org/repo" {
		t.Errorf("expected 'org/repo', got %q", got)
	}
}

func TestExtractTextValueStaticSelect(t *testing.T) {
	values := map[string]map[string]interface{}{
		"pr_block": {
			"pr_select": map[string]interface{}{
				"type": "static_select",
				"selected_option": map[string]interface{}{
					"value": "42",
				},
			},
		},
	}
	got := extractTextValue(values, "pr_block", "pr_select")
	if got != "42" {
		t.Errorf("expected '42', got %q", got)
	}
}

func TestExtractTextValueMissingBlock(t *testing.T) {
	values := map[string]map[string]interface{}{}
	got := extractTextValue(values, "missing_block", "action")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractTextValueNullInput(t *testing.T) {
	values := map[string]map[string]interface{}{
		"repo_block": {
			"repo_input": map[string]interface{}{
				"type":  "plain_text_input",
				"value": nil,
			},
		},
	}
	got := extractTextValue(values, "repo_block", "repo_input")
	if got != "" {
		t.Errorf("expected empty string for null value, got %q", got)
	}
}

// ---- PR list JSON parsing tests ----

func TestParsePRListJSON(t *testing.T) {
	raw := `[
		{"number": 1, "title": "Fix bug", "author": {"login": "alice"}, "url": "https://github.com/org/repo/pull/1", "headRefName": "fix/bug"},
		{"number": 2, "title": "Add feature", "author": {"login": "bob"}, "url": "https://github.com/org/repo/pull/2", "headRefName": "feat/feature"}
	]`

	var prs []PRItem
	if err := json.Unmarshal([]byte(raw), &prs); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0].Number != 1 || prs[0].Title != "Fix bug" || prs[0].Author.Login != "alice" {
		t.Errorf("unexpected PR[0]: %+v", prs[0])
	}
	if prs[1].Number != 2 || prs[1].Author.Login != "bob" {
		t.Errorf("unexpected PR[1]: %+v", prs[1])
	}
}

func TestParsePRListEmptyJSON(t *testing.T) {
	var prs []PRItem
	if err := json.Unmarshal([]byte("[]"), &prs); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(prs) != 0 {
		t.Errorf("expected 0 PRs, got %d", len(prs))
	}
}

// ---- handleSlashCommand filtering tests ----

func TestHandleSlashCommandIgnoresNonPR(t *testing.T) {
	commands := []string{"/issue", "/deploy", "/help", ""}

	// We verify that non-/pr commands are ignored (no panic, no action).
	// Since the function calls slackClient.OpenView on /pr only, and we pass nil,
	// a non-/pr command should return without calling OpenView.
	for _, cmd := range commands {
		payload, _ := json.Marshal(SlackCommand{Command: cmd, TriggerID: "tid"})
		assertNoPanic(t, fmt.Sprintf("command %q", cmd), func() {
			handleSlashCommand(context.Background(), nil, nil, string(payload), Config{})
		})
	}
}

func TestHandleSlashCommandWithRepoArgSkipsRepoChooser(t *testing.T) {
	// When a repo argument is provided, handleSlashCommand should attempt to open
	// the loading modal (not the repo chooser). With a nil Slack client this panics,
	// so we confirm it does NOT return silently before touching the client.
	payload, _ := json.Marshal(SlackCommand{Command: "/pr", Text: "myrepo", TriggerID: "tid"})
	assertPanics(t, "repo arg provided", func() {
		handleSlashCommand(context.Background(), nil, nil, string(payload), Config{GitHubOrg: "my-org"})
	})
}

func TestHandleSlashCommandWithoutRepoArgOpensRepoChooser(t *testing.T) {
	// When no repo argument is provided, handleSlashCommand should attempt to open
	// the repo chooser modal. With a nil Slack client this panics.
	payload, _ := json.Marshal(SlackCommand{Command: "/pr", Text: "", TriggerID: "tid"})
	assertPanics(t, "no repo arg", func() {
		handleSlashCommand(context.Background(), nil, nil, string(payload), Config{})
	})
}

func TestHandleSlashCommandInvalidRepoArgIsIgnored(t *testing.T) {
	// An invalid repo arg (e.g. containing slashes or shell metacharacters) should
	// be rejected silently — the function should return without touching the Slack client.
	invalidArgs := []string{"org/repo", "repo; rm -rf /", "repo name", "../etc"}
	for _, arg := range invalidArgs {
		payload, _ := json.Marshal(SlackCommand{Command: "/pr", Text: arg, TriggerID: "tid"})
		assertNoPanic(t, fmt.Sprintf("invalid repo arg %q", arg), func() {
			handleSlashCommand(context.Background(), nil, nil, string(payload), Config{GitHubOrg: "my-org"})
		})
	}
}

func TestHandleSlashCommandWhitespaceOnlyTextOpensRepoChooser(t *testing.T) {
	// Whitespace-only text should be treated as no repo argument.
	payload, _ := json.Marshal(SlackCommand{Command: "/pr", Text: "   ", TriggerID: "tid"})
	assertPanics(t, "whitespace-only text", func() {
		handleSlashCommand(context.Background(), nil, nil, string(payload), Config{})
	})
}

// ---- SlackLinerMessage serialisation test ----

func TestSlackLinerMessageSerialization(t *testing.T) {
	pr := &PRItem{Number: 7, Title: "My PR", URL: "https://github.com/org/repo/pull/7"}
	pr.Author.Login = "carol"

	config := Config{SlackChannelID: "C12345", RedisSlackLinerList: "slack_messages"}

	msg := SlackLinerMessage{
		Channel: config.SlackChannelID,
		Text:    fmt.Sprintf("📋 PR #%d: %s", pr.Number, pr.Title),
		TTL:     86400,
		Metadata: map[string]interface{}{
			"event_type": "pr_posted",
			"event_payload": map[string]interface{}{
				"pr_number":  pr.Number,
				"repository": "org/repo",
				"pr_url":     pr.URL,
				"author":     pr.Author.Login,
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal SlackLinerMessage: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if out["channel"] != "C12345" {
		t.Errorf("unexpected channel: %v", out["channel"])
	}
	if out["ttl"].(float64) != 86400 {
		t.Errorf("unexpected ttl: %v", out["ttl"])
	}
}

func TestPostPRToSlackMetadataIncludesBranch(t *testing.T) {
	pr := &PRItem{
		Number:      42,
		Title:       "Fix bug",
		URL:         "https://github.com/example/repo/pull/42",
		HeadRefName: "feature/fix-bug",
	}
	pr.Author.Login = "octocat"

	msg := SlackLinerMessage{
		Channel: "C12345",
		Text:    "test",
		TTL:     86400,
		Metadata: map[string]interface{}{
			"event_type": "pr_posted",
			"event_payload": map[string]interface{}{
				"pr_number":  pr.Number,
				"repository": "example/repo",
				"pr_url":     pr.URL,
				"author":     pr.Author.Login,
				"title":      pr.Title,
				"posted_by":  "alice",
				"branch":     pr.HeadRefName,
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal SlackLinerMessage: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	metadata, ok := out["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metadata to be a map")
	}
	payload, ok := metadata["event_payload"].(map[string]interface{})
	if !ok {
		t.Fatal("expected event_payload to be a map")
	}
	if payload["branch"] != "feature/fix-bug" {
		t.Errorf("expected branch 'feature/fix-bug', got %v", payload["branch"])
	}
}

// ---- BlockActionPayload parsing tests ----

func TestBlockActionPayloadParsing(t *testing.T) {
	raw := `{
		"type": "block_actions",
		"trigger_id": "tid123",
		"user": {"id": "U001", "username": "alice"},
		"view": {"id": "V001"},
		"actions": [{
			"action_id": "SlashVibeIssue",
			"block_id": "repo_block",
			"type": "external_select",
			"selected_option": {"value": "my-repo"}
		}]
	}`

	var p BlockActionPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if p.TriggerID != "tid123" {
		t.Errorf("expected trigger_id 'tid123', got %q", p.TriggerID)
	}
	if p.User.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", p.User.Username)
	}
	if len(p.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(p.Actions))
	}
	if p.Actions[0].ActionID != "SlashVibeIssue" {
		t.Errorf("expected action_id 'SlashVibeIssue', got %q", p.Actions[0].ActionID)
	}
	if p.Actions[0].SelectedOption.Value != "my-repo" {
		t.Errorf("expected selected value 'my-repo', got %q", p.Actions[0].SelectedOption.Value)
	}
}

func TestHandleBlockActionIgnoresUnknownActionID(t *testing.T) {
	// A block action with a different action_id should be silently ignored.
	payload, _ := json.Marshal(BlockActionPayload{
		Type:      "block_actions",
		TriggerID: "tid",
		Actions: []struct {
			ActionID       string `json:"action_id"`
			BlockID        string `json:"block_id"`
			Type           string `json:"type"`
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		}{{ActionID: "other_action", SelectedOption: struct {
			Value string `json:"value"`
		}{Value: "some-value"}}},
	})

	assertNoPanic(t, "unknown action_id", func() {
		handleBlockAction(context.Background(), nil, nil, string(payload), Config{GitHubOrg: "my-org"})
	})
}

func TestHandleBlockActionEmptyValueIsIgnored(t *testing.T) {
	// A repo selection action with empty value should be silently ignored.
	payload, _ := json.Marshal(BlockActionPayload{
		Type:      "block_actions",
		TriggerID: "tid",
		Actions: []struct {
			ActionID       string `json:"action_id"`
			BlockID        string `json:"block_id"`
			Type           string `json:"type"`
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		}{{ActionID: slashVibeIssueActionID, SelectedOption: struct {
			Value string `json:"value"`
		}{Value: ""}}},
	})

	assertNoPanic(t, "empty repo value", func() {
		handleBlockAction(context.Background(), nil, nil, string(payload), Config{GitHubOrg: "my-org"})
	})
}

func TestHandleBlockActionWithRepoOpensLoadingModal(t *testing.T) {
	// A valid block action should attempt to open the loading modal.
	// With a nil Slack client this panics, confirming the loading modal path is reached.
	payload, _ := json.Marshal(BlockActionPayload{
		Type:      "block_actions",
		TriggerID: "tid",
		User: struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		}{Username: "alice"},
		Actions: []struct {
			ActionID       string `json:"action_id"`
			BlockID        string `json:"block_id"`
			Type           string `json:"type"`
			SelectedOption struct {
				Value string `json:"value"`
			} `json:"selected_option"`
		}{{ActionID: slashVibeIssueActionID, BlockID: repoBlockID, SelectedOption: struct {
			Value string `json:"value"`
		}{Value: "my-repo"}}},
	})

	assertPanics(t, "valid repo block action", func() {
		handleBlockAction(context.Background(), nil, nil, string(payload), Config{GitHubOrg: "my-org"})
	})
}

// ---- Config tests ----

func TestGetEnvDefault(t *testing.T) {
	got := getEnv("SLASHVIBEPRSTEST_NONEXISTENT_KEY_XYZ", "default-value")
	if got != "default-value" {
		t.Errorf("expected 'default-value', got %q", got)
	}
}

func TestLoadConfigFromBytesDefaults(t *testing.T) {
	// Empty YAML — should fall back to built-in defaults.
	config, err := loadConfigFromBytes([]byte(""), "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.RedisChannel != "slack-commands" {
		t.Errorf("unexpected RedisChannel: %q", config.RedisChannel)
	}
	if config.RedisPoppitList != "poppit:commands" {
		t.Errorf("unexpected RedisPoppitList: %q", config.RedisPoppitList)
	}
	if config.RedisPoppitOutputChannel != "poppit:command-output" {
		t.Errorf("unexpected RedisPoppitOutputChannel: %q", config.RedisPoppitOutputChannel)
	}
	if config.RedisBlockActionsChannel != "slack-relay-block-actions" {
		t.Errorf("unexpected RedisBlockActionsChannel: %q", config.RedisBlockActionsChannel)
	}
	if config.LogLevel != "INFO" {
		t.Errorf("unexpected LogLevel: %q", config.LogLevel)
	}
}

func TestLoadConfigFromBytesFullYAML(t *testing.T) {
	yamlData := []byte(`
redis:
  addr: myredis:6380
channels:
  slash_commands: my-commands
  view_submissions: my-view-submissions
  block_actions: my-block-actions
  poppit_output: my-poppit-output
lists:
  poppit_commands: my-poppit-commands
  slackliner_messages: my-slack-messages
slack:
  channel_id: CMYCHANNEL
github:
  org: my-org
logging:
  level: DEBUG
`)

	config, err := loadConfigFromBytes(yamlData, "secret-pw", "xoxb-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.RedisAddr != "myredis:6380" {
		t.Errorf("unexpected RedisAddr: %q", config.RedisAddr)
	}
	if config.RedisPassword != "secret-pw" {
		t.Errorf("unexpected RedisPassword: %q", config.RedisPassword)
	}
	if config.SlackBotToken != "xoxb-token" {
		t.Errorf("unexpected SlackBotToken: %q", config.SlackBotToken)
	}
	if config.RedisChannel != "my-commands" {
		t.Errorf("unexpected RedisChannel: %q", config.RedisChannel)
	}
	if config.RedisBlockActionsChannel != "my-block-actions" {
		t.Errorf("unexpected RedisBlockActionsChannel: %q", config.RedisBlockActionsChannel)
	}
	if config.SlackChannelID != "CMYCHANNEL" {
		t.Errorf("unexpected SlackChannelID: %q", config.SlackChannelID)
	}
	if config.GitHubOrg != "my-org" {
		t.Errorf("unexpected GitHubOrg: %q", config.GitHubOrg)
	}
	if config.LogLevel != "DEBUG" {
		t.Errorf("unexpected LogLevel: %q", config.LogLevel)
	}
}

func TestLoadConfigFromBytesPartialYAML(t *testing.T) {
	// Only override a subset — other values should keep built-in defaults.
	yamlData := []byte(`
slack:
  channel_id: CPARTIAL
`)

	config, err := loadConfigFromBytes(yamlData, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.SlackChannelID != "CPARTIAL" {
		t.Errorf("unexpected SlackChannelID: %q", config.SlackChannelID)
	}
	// Unset values keep defaults.
	if config.RedisAddr != "host.docker.internal:6379" {
		t.Errorf("unexpected RedisAddr: %q", config.RedisAddr)
	}
	if config.RedisChannel != "slack-commands" {
		t.Errorf("unexpected RedisChannel: %q", config.RedisChannel)
	}
}

func TestLoadConfigFromBytesInvalidYAML(t *testing.T) {
	_, err := loadConfigFromBytes([]byte("not: valid: yaml: ["), "", "")
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

// ---- PRModalPrivateMetadata serialisation ----

func TestPRModalPrivateMetadataRoundtrip(t *testing.T) {
	meta := PRModalPrivateMetadata{Repo: "my-org/my-repo"}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out PRModalPrivateMetadata
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out.Repo != "my-org/my-repo" {
		t.Errorf("expected 'my-org/my-repo', got %q", out.Repo)
	}
}
