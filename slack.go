package main

import (
	"fmt"

	"github.com/slack-go/slack"
)

const (
	repoModalCallbackID            = "select_pr_repo_modal"
	prModalCallbackID              = "select_pr_modal"
	importIssueRepoModalCallbackID = "select_issue_repo_modal"
	issueModalCallbackID           = "select_issue_modal"
	slashVibePRActionID            = "SlashVibePR"
	slashVibeImportIssueActionID   = "SlashVibeImportIssue"
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
					repoBlockID,
					&slack.SelectBlockElement{
						Type:     slack.OptTypeExternal,
						ActionID: slashVibePRActionID,
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

// createAutoPostedModal returns a modal confirming that a single PR was
// automatically posted to the channel without requiring the user to choose.
func createAutoPostedModal(pr *PRItem, repo string) slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "PR Posted",
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
						Text: fmt.Sprintf(":white_check_mark: Only one open pull request was found for `%s`.\n\n*PR #%d: %s* has been posted to the channel.", repo, pr.Number, pr.Title),
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

// createIssueRepoChooserModal returns a modal for the user to select a repository
// when using the /import-issue command. It uses an external select that dispatches
// a block_actions event (with slashVibeImportIssueActionID) on selection.
func createIssueRepoChooserModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: importIssueRepoModalCallbackID,
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
						Text: "Select a repository to list its open issues.",
					},
				},
				slack.NewActionBlock(
					repoBlockID,
					&slack.SelectBlockElement{
						Type:     slack.OptTypeExternal,
						ActionID: slashVibeImportIssueActionID,
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

// createIssueLoadingModal returns a transient modal shown while Poppit fetches issues.
func createIssueLoadingModal() slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Loading Issues...",
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
						Text: ":hourglass_flowing_sand: Fetching open issues, please wait...",
					},
				},
			},
		},
	}
}

// createIssueChooserModal returns a modal presenting a dropdown of open issues.
// privateMetadata is stored in the modal and retrieved on submission.
func createIssueChooserModal(issues []IssueItem, repo, privateMetadata string) slack.ModalViewRequest {
	options := make([]*slack.OptionBlockObject, 0, len(issues))
	for _, issue := range issues {
		text := fmt.Sprintf("#%d: %s", issue.Number, issue.Title)
		if len(text) > 75 {
			text = text[:72] + "..."
		}
		options = append(options, &slack.OptionBlockObject{
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: text,
			},
			Value: fmt.Sprintf("%d", issue.Number),
		})
	}

	return slack.ModalViewRequest{
		Type:            slack.VTModal,
		CallbackID:      issueModalCallbackID,
		PrivateMetadata: privateMetadata,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Select an Issue",
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
						Text: fmt.Sprintf("*%s* — select an issue to post to the channel.", repo),
					},
				},
				&slack.InputBlock{
					Type:    slack.MBTInput,
					BlockID: "issue_block",
					Label: &slack.TextBlockObject{
						Type: slack.PlainTextType,
						Text: "Issue",
					},
					Element: &slack.SelectBlockElement{
						Type:     slack.OptTypeStatic,
						ActionID: "issue_select",
						Placeholder: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "Choose an issue",
						},
						Options: options,
					},
				},
			},
		},
	}
}

// createIssueAutoPostedModal returns a modal confirming that a single issue was
// automatically posted to the channel without requiring the user to choose.
func createIssueAutoPostedModal(issue *IssueItem, repo string) slack.ModalViewRequest {
	return slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Issue Posted",
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
						Text: fmt.Sprintf(":white_check_mark: Only one open issue was found for `%s`.\n\n*Issue #%d: %s* has been posted to the channel.", repo, issue.Number, issue.Title),
					},
				},
			},
		},
	}
}
