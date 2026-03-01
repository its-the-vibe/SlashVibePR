package main

import (
	"fmt"

	"github.com/slack-go/slack"
)

const (
	repoModalCallbackID    = "select_pr_repo_modal"
	prModalCallbackID      = "select_pr_modal"
	slashVibeIssueActionID = "SlashVibeIssue"
)

// createRepoChooserModal returns a modal for the user to select a repository
// from a dropdown populated by OctoCatalog (external select).
// The select element is placed in an actions block so that choosing a repo
// immediately dispatches a block_actions event (no submit button required),
// which provides a fresh trigger_id and prevents the PR modal from being missed.
func createRepoChooserModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: repoModalCallbackID,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Select Repository",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				&slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: "Select a repository to list its open pull requests.",
					},
				},
				slack.NewActionBlock(
					"repo_block",
					&slack.SelectBlockElement{
						Type:     slack.OptTypeExternal,
						ActionID: slashVibeIssueActionID,
						Placeholder: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "Search for a repo...",
						},
					},
				),
			},
		},
	}
}

// createLoadingModal returns a transient modal shown while Poppit fetches PRs.
func createLoadingModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Loading PRs...",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				&slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: ":hourglass_flowing_sand: Fetching open pull requests, please wait...",
					},
				},
			},
		},
	}
}

// createPRChooserModal returns a modal presenting a dropdown of open PRs.
// privateMetadata is stored in the modal and retrieved on submission.
func createPRChooserModal(prs []PRItem, repo, privateMetadata string) slack.ModalViewRequest {
	options := make([]*slack.OptionBlockObject, 0, len(prs))
	for _, pr := range prs {
		text := fmt.Sprintf("#%d: %s", pr.Number, pr.Title)
		if len(text) > 75 {
			text = text[:72] + "..."
		}
		options = append(options, &slack.OptionBlockObject{
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: text,
			},
			Value: fmt.Sprintf("%d", pr.Number),
		})
	}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      prModalCallbackID,
		PrivateMetadata: privateMetadata,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Select a Pull Request",
		},
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Post to Channel",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				&slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: fmt.Sprintf("*%s* — select a pull request to post to the channel.", repo),
					},
				},
				&slack.InputBlock{
					Type:    slack.MBTInput,
					BlockID: "pr_block",
					Label: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Pull Request",
					},
					Element: &slack.SelectBlockElement{
						Type:     slack.OptTypeStatic,
						ActionID: "pr_select",
						Placeholder: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "Choose a pull request",
						},
						Options: options,
					},
				},
			},
		},
	}
}

// createErrorModal returns a modal displaying an error message.
func createErrorModal(message string) slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Error",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Close",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				&slack.SectionBlock{
					Type: slack.MBTSection,
					Text: &slack.TextBlockObject{
						Type: slack.MarkdownType,
						Text: fmt.Sprintf(":x: %s", message),
					},
				},
			},
		},
	}
}
