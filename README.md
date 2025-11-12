# issue-miner
`issue-miner` is a GitHub CLI (`gh`) extension written in Go for analyzing GitHub issues, providing insights through metrics and relationship graphs.

Example usage:

```shell
# Output stats about 100 issues in a specified repository
gh issue-miner pulse --repo octocat/Hello-World --limit 100

# Fetch 20 issues from a repository
gh issue-miner fetch --repo octocat/Hello-World --limit 20

# Graph an issue and its references up to depth 2, allowing cross-repo links
gh issue-miner graph https://github.com/octocat/Hello-World/issues/349 --depth 2 --cross-repo
```

## Subcommands
`issue-miner` has several subcommands to perform different tasks. Here they are, along with the filters they imply:

Subcommand | Default Filters | Description
---        | ---             | ---
fetch      | --limit 100     | List issues and their basic details
pulse      | --limit 100     | Show pulse metrics about issues
graph      | --limit 100 --depth 1 --max-nodes 500 | Graph issues and links in/out

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


## Filters vs Options

`issue-miner` distinguishes between filters and options so it's clear what affects issue selection vs processing/output.

- Filters: narrow which issues/PRs are selected for analysis. Examples: `--repo`, `--limit`, `--labels`, `--state`, `--include-prs`, and a positional issue `<url>`.
- Options: control how selected issues are processed or how results are presented (for example `--depth`, `--max-nodes`, `--cross-repo`, `--format`, `--sort`).

Below are the common filters with their defaults:

Filter   | Default | Description
---      | ---     | ---
`<url>`    |         | URL of an issue to analyze (all other filters will be ignored)
`--repo`   | `origin` remote | NWO or URL of the repository to analyze
`--limit`  | 100     | Maximum number of issues to select
`--include-prs` | true | Include pull requests in the selection
`--labels` |      | Select issues with these labels (comma-separated)
`--state`  | open    | Select issues with this state (open, closed)
`--created`   |  | Issues created within this time frame<br />(e.g., `30d`, `90d..60d`, `2025-02-01..`)
`--updated`   |  | Issues updated within this time frame<br />(e.g., `30d`, `90d..60d`, `2025-02-01..`)
`--closed`    |  | Issues closed within this time frame<br />(e.g., `30d`, `90d..60d`, `2025-02-01..`)

Options that change processing or output (not selection):

Option       | Default  | Description
---          | ---      | ---
`--depth`      | 1        | Traversal depth when graphing references (affects processing only)
`--max-nodes`  | 500      | Maximum number of nodes to visit during graph traversal (0 = unlimited)
`--cross-repo` | false    | Allow following references across repositories when recursing (processing option)
`--format`     | text     | Output format (`text`, `json`, `dot`)
`--sort`       | created_at | Sort field for output
`--order`      | desc     | Sort order

Important: when running the `graph` command, filters affect only the initial issue selection (the set of starting issues). The graph traversal/expansion step is controlled by options such as `--depth` and `--cross-repo` and may discover and include additional issues that were not part of the initial filtered set.

## Examples: Time-based filters
Here are a couple of examples showing how to use the new time filters.

- Fetch issues created on or after a date (single-day selection uses the whole day):

```bash
# issues created on 2025-11-01 (local date interpreted as UTC day)
gh issue-miner fetch --repo owner/repo --created 2025-11-01
```

- Fetch issues created between two dates (inclusive):

```bash
# issues created from 2025-10-01 through 2025-10-31 (inclusive)
gh issue-miner fetch --repo owner/repo --created 2025-10-01..2025-10-31
```

- Fetch issues updated within the last 30 days:

```bash
gh issue-miner pulse --repo owner/repo --updated 30d
```

- Fetch issues created after a date (open-ended range):

```bash
# issues created on/after 2025-11-02 (use open-ended range up to now)
gh issue-miner fetch --repo owner/repo --created 2025-11-02..
```

Notes:
- Dates use `YYYY-MM-DD` and are interpreted as that UTC day (start at 00:00 UTC).
- Ranges are inclusive of the start date and inclusive of the end date (implemented as end-of-day).
- Relative forms like `7d` mean "last 7 days" (from now back 7×24h).

<!--
PHASE 3:
--assignee  | all | Issues assigned to this user
--author    | all | Issues created by this author
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
