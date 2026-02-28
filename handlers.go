package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/slack-go/slack"
)

const (
	poppitPRListType   = "slash-vibe-pr-list"
	prSessionKeyTTL    = time.Hour
	prSessionKeyPrefix = "slashvibeprs:"
	defaultPRLimit     = 50
)

// subscribeToSlashCommands subscribes to the Redis slash-commands channel and
// dispatches any /pr command to handleSlashCommand.
func subscribeToSlashCommands(ctx context.Context, rdb *redis.Client, slackClient *slack.Client, config Config) {
	pubsub := rdb.Subscribe(ctx, config.RedisChannel)
	defer pubsub.Close()

	Info("Subscribed to Redis channel: %s", config.RedisChannel)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			handleSlashCommand(ctx, slackClient, msg.Payload, config)
		}
	}
}

// handleSlashCommand processes a raw slash command payload. Only /pr is handled;
// all other commands are silently ignored.
func handleSlashCommand(ctx context.Context, slackClient *slack.Client, payload string, config Config) {
	var cmd SlackCommand
	if err := json.Unmarshal([]byte(payload), &cmd); err != nil {
		Error("Error unmarshaling slash command: %v", err)
		return
	}

	if cmd.Command != "/pr" {
		return
	}

	Info("Received /pr command from user %s", cmd.UserName)

	modal := createRepoChooserModal()
	var viewResp *slack.ViewResponse
	var err error
	if viewResp, err = slackClient.OpenView(cmd.TriggerID, modal); err != nil {
		Error("Error opening repo chooser modal: %v", err)
		return
	}

	Debug("Repo chooser modal opened successfully with view_id: %s", viewResp.ID)
}

// subscribeToViewSubmissions subscribes to the Redis view-submission channel and
// routes each submission to the appropriate handler based on callback_id.
func subscribeToViewSubmissions(ctx context.Context, rdb *redis.Client, slackClient *slack.Client, config Config) {
	pubsub := rdb.Subscribe(ctx, config.RedisViewSubmissionChannel)
	defer pubsub.Close()

	Info("Subscribed to Redis channel: %s", config.RedisViewSubmissionChannel)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			handleViewSubmission(ctx, rdb, slackClient, msg.Payload, config)
		}
	}
}

// handleViewSubmission decodes a view submission and routes it by callback_id.
func handleViewSubmission(ctx context.Context, rdb *redis.Client, slackClient *slack.Client, payload string, config Config) {
	var submission ViewSubmission
	if err := json.Unmarshal([]byte(payload), &submission); err != nil {
		Error("Error unmarshaling view submission: %v", err)
		return
	}

	switch submission.View.CallbackID {
	case repoModalCallbackID:
		handleRepoSelection(ctx, rdb, slackClient, submission, config)
	case prModalCallbackID:
		handlePRSelection(ctx, rdb, submission, config)
	}
}

// handleRepoSelection processes the repo-chooser modal submission:
//  1. Pushes the loading modal (replacing the repo-chooser modal).
//  2. Sends a Poppit command to run `gh pr list`.
func handleRepoSelection(ctx context.Context, rdb *redis.Client, slackClient *slack.Client, submission ViewSubmission, config Config) {
	repoName := extractTextValue(submission.View.State.Values, "repo_block", slashVibeIssueActionID)
	if repoName == "" {
		Warn("Repo selection submission has empty repo")
		return
	}

	repo := config.GitHubOrg + "/" + repoName

	Info("User %s selected repo: %s", submission.User.Username, repo)

	// Update the modal to show loading status (replaces the repo chooser modal).
	// Use UpdateView instead of PushView to avoid trigger_id expiration issues with Redis queuing.
	loadingModal := createLoadingModal()
	_, err := slackClient.UpdateView(loadingModal, "", "", submission.View.ID)
	if err != nil {
		Error("Error updating modal to loading state: %v", err)
		return
	}

	viewID := submission.View.ID
	Debug("Loading modal opened with view_id: %s", viewID)

	if err := sendPRListCommand(ctx, rdb, repo, viewID, submission.User.Username, config); err != nil {
		Error("Error sending Poppit command for repo %s: %v", repo, err)
	}
}

// sendPRListCommand pushes a Poppit command to list open PRs for the given repo.
// The view_id is passed in metadata so handlePoppitOutput can update the correct modal.
func sendPRListCommand(ctx context.Context, rdb *redis.Client, repo, viewID, username string, config Config) error {
	cmd := fmt.Sprintf(
		"gh pr list --repo %s --json number,title,author,url,headRefName --limit %d",
		repo, defaultPRLimit,
	)

	poppitCmd := PoppitCommand{
		Repo:     repo,
		Branch:   "",
		Type:     poppitPRListType,
		Dir:      "/tmp",
		Commands: []string{cmd},
		Metadata: map[string]interface{}{
			"view_id":  viewID,
			"repo":     repo,
			"username": username,
		},
	}

	payload, err := json.Marshal(poppitCmd)
	if err != nil {
		return fmt.Errorf("failed to marshal Poppit command: %w", err)
	}

	if err := rdb.RPush(ctx, config.RedisPoppitList, payload).Err(); err != nil {
		return fmt.Errorf("failed to push Poppit command to Redis: %w", err)
	}

	return nil
}

// handlePRSelection processes the PR-chooser modal submission:
//  1. Looks up PR details stored in Redis by the view ID.
//  2. Posts the selected PR to the configured Slack channel via SlackLiner.
func handlePRSelection(ctx context.Context, rdb *redis.Client, submission ViewSubmission, config Config) {
	prNumber := extractTextValue(submission.View.State.Values, "pr_block", "pr_select")
	if prNumber == "" {
		Warn("PR selection submission has empty PR number")
		return
	}

	// Retrieve PR list from Redis using the view ID as the session key.
	sessionKey := prSessionKeyPrefix + submission.View.ID
	prJSON, err := rdb.Get(ctx, sessionKey).Result()
	if err != nil {
		Error("Error fetching PR session data from Redis (key=%s): %v", sessionKey, err)
		return
	}

	var prs []PRItem
	if err := json.Unmarshal([]byte(prJSON), &prs); err != nil {
		Error("Error parsing PR session data: %v", err)
		return
	}

	// Parse private_metadata to get the repo name.
	var meta PRModalPrivateMetadata
	if err := json.Unmarshal([]byte(submission.View.PrivateMetadata), &meta); err != nil {
		Error("Error parsing private metadata: %v", err)
		return
	}

	// Find the selected PR by number.
	var selectedPR *PRItem
	for i := range prs {
		if fmt.Sprintf("%d", prs[i].Number) == prNumber {
			selectedPR = &prs[i]
			break
		}
	}

	if selectedPR == nil {
		Warn("Could not find PR #%s in session data", prNumber)
		return
	}

	Info("User %s selected PR #%d from %s", submission.User.Username, selectedPR.Number, meta.Repo)

	if err := postPRToSlack(ctx, rdb, selectedPR, meta.Repo, submission.User.Username, config); err != nil {
		Error("Error posting PR to Slack: %v", err)
		return
	}

	// Clean up the session key.
	if err := rdb.Del(ctx, sessionKey).Err(); err != nil {
		Warn("Failed to delete PR session key %s: %v", sessionKey, err)
	}

	Info("PR #%d from %s posted to Slack channel", selectedPR.Number, meta.Repo)
}

// postPRToSlack pushes a formatted PR message to the SlackLiner Redis list.
func postPRToSlack(ctx context.Context, rdb *redis.Client, pr *PRItem, repo, postedBy string, config Config) error {
	messageText := fmt.Sprintf(
		"📋 *Pull Request shared by @%s*\n\n"+
			"*Repository:* %s\n"+
			"*PR #%d:* %s\n"+
			"*Author:* %s\n"+
			"*Link:* <%s|View PR>",
		postedBy,
		repo,
		pr.Number,
		pr.Title,
		pr.Author.Login,
		pr.URL,
	)

	msg := SlackLinerMessage{
		Channel: config.SlackChannelID,
		Text:    messageText,
		TTL:     86400,
		Metadata: map[string]interface{}{
			"event_type": "pr_posted",
			"event_payload": map[string]interface{}{
				"pr_number":  pr.Number,
				"repository": repo,
				"pr_url":     pr.URL,
				"author":     pr.Author.Login,
				"title":      pr.Title,
				"posted_by":  postedBy,
				"branch":     pr.HeadRefName,
			},
		},
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal SlackLiner message: %w", err)
	}

	if err := rdb.RPush(ctx, config.RedisSlackLinerList, payload).Err(); err != nil {
		return fmt.Errorf("failed to push message to SlackLiner list: %w", err)
	}

	return nil
}

// subscribeToPoppitOutput subscribes to the Poppit command-output channel and
// handles PR list results.
func subscribeToPoppitOutput(ctx context.Context, rdb *redis.Client, slackClient *slack.Client, config Config) {
	pubsub := rdb.Subscribe(ctx, config.RedisPoppitOutputChannel)
	defer pubsub.Close()

	Info("Subscribed to Redis channel: %s", config.RedisPoppitOutputChannel)

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			handlePoppitOutput(ctx, rdb, slackClient, msg.Payload, config)
		}
	}
}

// handlePoppitOutput processes a Poppit output event for slash-vibe-pr-list:
//  1. Parses the PR list from stdout.
//  2. Stores the PRs in Redis keyed by the view ID.
//  3. Updates the loading modal to display the PR chooser.
func handlePoppitOutput(ctx context.Context, rdb *redis.Client, slackClient *slack.Client, payload string, config Config) {
	var output PoppitOutput
	if err := json.Unmarshal([]byte(payload), &output); err != nil {
		Error("Error unmarshaling Poppit output: %v", err)
		return
	}

	if output.Type != poppitPRListType {
		return
	}

	Debug("Received Poppit PR list output")

	metadata := output.Metadata
	if metadata == nil {
		Warn("No metadata in Poppit PR list output")
		return
	}

	viewID, _ := metadata["view_id"].(string)
	repo, _ := metadata["repo"].(string)
	username, _ := metadata["username"].(string)

	if viewID == "" || repo == "" {
		Warn("Missing view_id or repo in Poppit output metadata")
		return
	}

	// Parse the PR list from Poppit stdout.
	var prs []PRItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(output.Output)), &prs); err != nil {
		Error("Error parsing PR list JSON for repo %s: %v", repo, err)
		updateModalWithErrorByID(slackClient, viewID, "Failed to parse the pull request list. Please try again.")
		return
	}

	if len(prs) == 0 {
		Info("No open PRs found for repo %s (user: %s)", repo, username)
		updateModalWithErrorByID(slackClient, viewID, fmt.Sprintf("No open pull requests found for `%s`.", repo))
		return
	}

	Info("Found %d open PRs for repo %s (user: %s)", len(prs), repo, username)

	// Store the PR list in Redis so handlePRSelection can retrieve it.
	prJSON, err := json.Marshal(prs)
	if err != nil {
		Error("Error marshaling PR list for Redis: %v", err)
		return
	}

	sessionKey := prSessionKeyPrefix + viewID
	if err := rdb.Set(ctx, sessionKey, prJSON, prSessionKeyTTL).Err(); err != nil {
		Error("Error storing PR session in Redis (key=%s): %v", sessionKey, err)
		return
	}

	// Build private_metadata for the PR chooser modal.
	meta := PRModalPrivateMetadata{Repo: repo}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		Error("Error marshaling PR modal metadata: %v", err)
		return
	}

	// Replace the loading modal with the PR chooser.
	// Use empty hash to skip Slack's optimistic lock check, avoiding stale hash issues.
	prModal := createPRChooserModal(prs, repo, string(metaJSON))
	if _, err := slackClient.UpdateView(prModal, "", "", viewID); err != nil {
		Error("Error updating modal with PR list: %v", err)
		return
	}

	Debug("PR chooser modal updated successfully for view_id: %s", viewID)
}

// updateModalWithErrorByID replaces the current modal content with an error message.
// It uses an empty hash to skip Slack's optimistic lock check, avoiding stale hash issues.
func updateModalWithErrorByID(slackClient *slack.Client, viewID, message string) {
	if _, err := slackClient.UpdateView(createErrorModal(message), "", "", viewID); err != nil {
		Error("Error updating modal with error message: %v", err)
	}
}

// updateModalWithError replaces the current modal content with an error message using the provided hash.
// DEPRECATED: Use updateModalWithErrorByID instead to avoid hash staleness issues.
func updateModalWithError(slackClient *slack.Client, viewID, viewHash, message string) {
	if _, err := slackClient.UpdateView(createErrorModal(message), "", viewHash, viewID); err != nil {
		Error("Error updating modal with error message: %v", err)
	}
}

// extractTextValue returns the string value for a given blockID/actionID from
// a Slack view state. It handles both plain-text inputs and static selects.
func extractTextValue(values map[string]map[string]interface{}, blockID, actionID string) string {
	block, ok := values[blockID]
	if !ok {
		return ""
	}

	action, ok := block[actionID]
	if !ok {
		return ""
	}

	actionMap, ok := action.(map[string]interface{})
	if !ok {
		return ""
	}

	// Static select: selected_option.value
	if selectedOption, ok := actionMap["selected_option"].(map[string]interface{}); ok {
		if value, ok := selectedOption["value"].(string); ok {
			return value
		}
	}

	// Plain text input: value
	if value, ok := actionMap["value"].(string); ok {
		return value
	}

	return ""
}
