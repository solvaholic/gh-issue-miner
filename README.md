# issue-miner
`issue-miner` is a tool for deriving insights from data about GitHub issues.

Example usage:

```shell
# Summarize open bug reports
issue-miner summarize --repo octocat/Hello-World --labels bug

# Graph links into and out of open issues
issue-miner graph --repo octocat/Hello-World

# Graph links into and out of a specific issue
issue-miner graph https://github.com/octocat/Hello-World/issues/42
```

## Subcommands
`issue-miner` has several subcommands to perform different tasks. Here they are, along with the filters they imply:

Subcommand | Filters      | Description
---        | ---          | ---
pulse      | --state open | Show pulse metrics about issues
graph      | --state open | Graph issues and links in/out

<!--
summarize | --state open | Summarize issues in a repository
-->

## Filters
`issue-miner` supports a variety of filters to narrow down the issues you want to analyze. Here they are, with their default values:

Filter   | Default | Description
---      | ---     | ---
<url>    |         | URL of an issue to analyze (all other filters will be ignored)
--repo   | $(git remote get-url --push origin) | NWO or URL of the repository to analyze
--labels | all     | Issues with these labels (comma-separated)
--state  | open    | Issues with this state or status (open, closed, all)
--limit  | 100     | Maximum number of issues to return

<!--
--created   | all | Issues created within this time frame<br />(e.g., `30days`, `90-60days`, `2025-02-01`)
--updated   | all | Issues updated within this time frame<br />(e.g., `30days`, `90-60days`, `2025-02-01`)
--closed    | all | Issues closed within this time frame<br />(e.g., `30days`, `90-60days`, `2025-02-01`)
--assignee  | all | Issues assigned to this user
--author    | all | Issues created by this author
--commenter | all | Issues commented on by this user
--mention   | all | Issues mentioning this user
--milestone | all | Issues in this milestone
-->

## Output Options
`issue-miner` supports various output options to suit your needs. Here are the available options:

Option      | Default    | Description
---         | ---        | ---
--output    | stdout     | Output file (default is stdout)
--sort      | created_at | Sort issues by this field (e.g., `created_at`, `updated_at`, `comments`)
--direction | desc       | Sort direction (asc or desc)

<!--
--center | issues | Center groupings on issues, users, or labels
-->