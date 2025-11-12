# issue-miner
`issue-miner` is a GitHub CLI (`gh`) extension written in Go for analyzing GitHub issues, providing insights through metrics and relationship graphs.

Example usage:

```shell
# Output stats about 100 issues in a specified repository
gh issue-miner pulse --repo octocat/Hello-World --limit 100

# Fetch 20 issues from a repository
gh issue-miner fetch --repo octocat/Hello-World --limit 20
```

## Subcommands
`issue-miner` has several subcommands to perform different tasks. Here they are, along with the filters they imply:

Subcommand | Default Filters | Description
---        | ---             | ---
fetch      | --limit 100     | List issues and their basic details
pulse      | --limit 100     | Show pulse metrics about issues

<!--
PHASE 2:
graph      | --limit 100     | Graph issues and links in/out
-->

<!--
FUTURE?:
summarize | --state open | Summarize issues in a repository
-->
## Installation

Install from GitHub (recommended):

```bash
gh extension install solvaholic/gh-issue-miner
```

Local development: see `DEVELOPER.md` for local install and testing instructions (`gh extension install .`).


## Filters
`issue-miner` supports a variety of filters to narrow down the issues you want to analyze. Here they are, with their default values:

Filter   | Default | Description
---      | ---     | ---
<url>    |         | URL of an issue to analyze (all other filters will be ignored)
--repo   | `origin` remote | NWO or URL of the repository to analyze
--limit  | 100     | Maximum number of issues to return

<!--
PHASE 3:
--labels | all     | Issues with these labels (comma-separated)
--state  | open    | Issues with this state or status (open, closed, all)
--assignee  | all | Issues assigned to this user
--author    | all | Issues created by this author
--created   | all | Issues created within this time frame<br />(e.g., `30days`, `90-60days`, `2025-02-01`)
--updated   | all | Issues updated within this time frame<br />(e.g., `30days`, `90-60days`, `2025-02-01`)
--closed    | all | Issues closed within this time frame<br />(e.g., `30days`, `90-60days`, `2025-02-01`)
-->

<!--
FUTURE?:
--commenter | all | Issues commented on by this user
--mention   | all | Issues mentioning this user
--milestone | all | Issues in this milestone
-->

<!--
PHASE 3:

## Output Options
`issue-miner` supports various output options to suit your needs. Here are the available options:

Option      | Default    | Description
---         | ---        | ---
--sort      | created_at | Sort issues by this field (e.g., `created_at`, `updated_at`)
--order     | desc       | Sort order (asc or desc)
--output    | stdout     | Output file (default is stdout)
-->

<!--
FUTURE?:
--center | issues | Center groupings on issues, users, or labels
-->
