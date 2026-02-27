package main

// SlackCommand represents an incoming Slack slash command payload.
type SlackCommand struct {
	Command     string `json:"command"`
	Text        string `json:"text"`
	ResponseURL string `json:"response_url"`
	TriggerID   string `json:"trigger_id"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	ChannelID   string `json:"channel_id"`
}

// ViewSubmission represents a Slack view submission event payload.
type ViewSubmission struct {
	Type      string `json:"type"`
	TriggerID string `json:"trigger_id"`
	View      struct {
		ID              string `json:"id"`
		Hash            string `json:"hash"`
		CallbackID      string `json:"callback_id"`
		PrivateMetadata string `json:"private_metadata"`
		State           struct {
			Values map[string]map[string]interface{} `json:"values"`
		} `json:"state"`
	} `json:"view"`
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

// PoppitCommand is the payload sent to Poppit via Redis to execute a command.
type PoppitCommand struct {
	Repo     string                 `json:"repo"`
	Branch   string                 `json:"branch"`
	Type     string                 `json:"type"`
	Dir      string                 `json:"dir"`
	Commands []string               `json:"commands"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PoppitOutput is the payload published by Poppit after command execution.
type PoppitOutput struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Type     string                 `json:"type"`
	Command  string                 `json:"command"`
	Output   string                 `json:"output"`
}

// SlackLinerMessage is the payload pushed to SlackLiner for posting to Slack.
type SlackLinerMessage struct {
	Channel  string                 `json:"channel"`
	Text     string                 `json:"text"`
	TTL      int                    `json:"ttl,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PRItem represents a single pull request returned by `gh pr list --json`.
type PRItem struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
}

// PRModalPrivateMetadata is stored in the PR-chooser modal's private_metadata field.
type PRModalPrivateMetadata struct {
	Repo string `json:"repo"`
}
