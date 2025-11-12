# Issue-Miner Specification

## Overview
`issue-miner` is a GitHub CLI (`gh`) extension written in Go for analyzing GitHub issues, providing insights through metrics and relationship graphs.

## Goals
1. Leverage GitHub CLI's built-in authentication and API access
2. Generate actionable insights from issue data
3. Visualize issue relationships and patterns
4. Provide a user-friendly command-line interface
5. Easy installation via `gh extension install`

## Non-Goals (Out of Scope for v1.0)
- Pull request analysis
- Real-time monitoring/webhooks
- Web dashboard or GUI
- Writing back to GitHub (no issue creation/modification)
- Standalone authentication (use `gh` auth)

## Core Features

### 1. GitHub CLI Extension
**Purpose**: Integrate seamlessly with GitHub CLI

**Requirements**:
- Install via: `gh extension install solvaholic/gh-issue-miner`
- Invoke via: `gh issue-miner <subcommand>`
- Inherit authentication from `gh auth` (no separate token needed)
- Use `gh` API client for all GitHub interactions

**Technical Approach**:
- Repository name: `gh-issue-miner` (GitHub CLI convention)
- Single binary built with Go
- Use `cli/go-gh` library for GitHub API calls
- No authentication code needed - `gh` handles it

### 2. CLI Framework
**Purpose**: Provide intuitive command-line interface

**Requirements**:
- Subcommands: `fetch`, `pulse`, `graph` (Phase 1 will include `fetch` and `pulse`)
- Global options for filtering and output
- Help text for all commands and options
- Input validation with clear error messages

**Technical Approach**:
- Use `spf13/cobra` library (same as `gh` CLI)
- Command structure:
  - `gh issue-miner pulse [flags]`
  - `gh issue-miner graph [flags] [issue-url]`

### 3. GitHub API Integration
**Purpose**: Fetch issue data from GitHub

**Requirements**:
- Use GitHub CLI's authenticated client
- Fetch issues from specified repository
- Support pagination for large result sets
- Handle API rate limits gracefully

**Technical Approach**:
- Use `cli/go-gh` package for API calls
- GraphQL API for efficient data fetching
- REST API fallback where needed

**API Queries**:
- List issues: GraphQL repository.issues query
- Get single issue: GraphQL issue query with references
- Parse issue body/comments for cross-references

### 4. Filtering System
**Purpose**: Allow users to narrow down issues for analysis

**Required Filters** (v1.0):
- `--repo <NWO>`: Repository in `owner/repo` format (default: current repo from git). Commands operate on the current repository when `--repo` is not provided.
- `--limit <n>`: Maximum number of issues (default: 100).

Note: For the Phase 1 release, only `--repo` and `--limit` are supported. `--state` and other filters are planned for later phases.

**Enhanced Filters** (Phase 3):
- `--label <labels>`: Comma-separated list of labels
- `--assignee <user>`: Filter by assignee
- `--author <user>`: Filter by author
- `--created <timeframe>`: Issues created within timeframe (e.g., `7d`, `30d`, `2024-01-01..2024-12-31`)
- `--closed <timeframe>`: Issues closed within timeframe (e.g., `7d`, `30d`, `2024-01-01..2024-12-31`)

**Future Filters** (post-v1.0):
- `--updated <timeframe>`: Issues updated within timeframe
- `--milestone <name>`: Filter by milestone
- `--mentioned <user>`: Issues mentioning user

**Technical Approach**:
- Build GraphQL query filters dynamically
- Validate filter combinations before querying
- Support time range parsing: relative (e.g., `7d`, `30d`) and absolute (ISO 8601 dates or ranges)

### 5. Fetch Command
**Purpose**: Fetch list of issues and their basic details

**Command**: `gh issue-miner fetch [flags]`

**Behavior**:
- Retrieve issues from the target repository (current repo or `--repo`) subject to supported filters.
- Phase 1: supports `--repo` and `--limit` only; other filters added in later phases.
- Supports pagination; stops after `--limit` issues are collected.
- Outputs a concise list with: issue number, state, title, labels, assignee, created_at, updated_at, comments_count.

**Output Format** (Phase 1):
```
#123  open  "Issue title"  bug,help wanted  alice  2025-01-02  2025-01-10  4
```

**Technical Approach**:
- Use `cli/go-gh` GraphQL queries to list issues and fields.
- Request minimal fields for Phase 1 for efficiency.
- Parse labels and assignee information for output.

### 6. Pulse Command
**Purpose**: Show metrics about repository issues, respecting all provided filters

**Command**: `gh issue-miner pulse [flags]`

**Behavior**:
- Calculate metrics for issues in the target repository (current repo or `--repo`) subject to supported filters.
- Useful for focused analysis when enhanced filters are available in later phases.
- In Phase 1 the `pulse` command accepts only `--repo` and `--limit`; additional filters (labels, author, time ranges) are added in later phases.

**Output Metrics**:
- Applied filters summary (if any non-default filters used)
- Total issue count (matching filters)
- Issue state breakdown (if `--state all`)
- Issues opened in last 7/30/90 days (from filtered set)
- Issues closed in last 7/30/90 days (from filtered set)
- Average time to close (for closed issues in filtered set)
- Most active issues (by comments, top 5)
- Label distribution (top 10, from filtered set)
- Assignee distribution (from filtered set)
- Author distribution (from filtered set, if multiple authors)

**Output Format** (MVP - Phase 1):
```
Repository: owner/repo

Issues:
  Open: 42
  Closed: 15
  Total: 57

Activity:
  Opened (7d/30d/90d): 5 / 15 / 42
  Closed (7d/30d/90d): 3 / 15 / 38
  Avg time to close: 12.5 days

Top Labels:
  bug: 12
  enhancement: 8
  documentation: 5
```

**Output Format** (Enhanced - Phase 3 with filters):
```
Repository: owner/repo
Filters: label=bug, author=octocat, created=30d

Issues (matching filters):
  Open: 12
  Closed: 8
  Total: 20

Activity (within filtered set):
  Opened (7d/30d/90d): 2 / 8 / 15
  Closed (7d/30d/90d): 1 / 8 / 12
  Avg time to close: 8.3 days

Most Active:
  #123 "Critical bug" (42 comments)
  #118 "Login issue" (28 comments)
  #105 "Performance problem" (15 comments)

Top Labels:
  bug: 20 (100%)
  security: 5 (25%)
  high-priority: 3 (15%)

Assignees:
  alice: 8 (40%)
  bob: 6 (30%)
  unassigned: 6 (30%)

Authors:
  octocat: 20 (100%)
```

**Technical Approach**:
- Phase 1 (MVP): Fetch all issues matching basic filters (--repo, --limit)
- Phase 3 (Enhanced): Apply all provided filters to GraphQL query
- Calculate all metrics from the filtered issue set
- Show filter summary when non-default filters are applied
- Format output using `text/tabwriter` for alignment

### 7. Graph Command
**Purpose**: Visualize issue relationships and cross-references

**Command**: `gh issue-miner graph [flags] [issue-url]`

**Behavior**:
- If `issue-url` provided: Graph that specific issue and its references
- Otherwise: Graph all issues matching filters

**Relationships to Track**:
- Issue-to-issue references (detected from issue URLs or textual references such as `#123` or `owner/repo#123` in bodies/comments)
- Cross-repository references (explicit `owner/repo#123` or full issue URLs that point to other repositories)
- "Closes/Fixes" relationships (explicit close keywords or timeline events where one issue closes another)
- Timeline-originated references (events and timeline items obtained from the GitHub timeline API)
- User-to-issue interactions:
  - Comments (user commented on an issue or PR)
  - Reviews (user added a review to a pull request)
  - Label modifications (user added or removed labels)
  - State changes (user opened, merged, closed, or reopened an issue or PR)
  - Project board events (user added or removed an issue from a project/column)
  - Other timeline actions (milestone changes, lock/unlock, transfers)
- Record actor, timestamp, and context/source (timeline event vs parsed body/comment) for each relationship
- Represent edges with metadata: action type, actor (username), timestamp, and source (timeline, webhook, or parsed text)

**Output Format**:
- Text-based graph (ASCII art) to stdout by default
- Optional: Export to DOT format (`--format dot`)
- Optional: JSON format (`--format json`)

**Example Output** (text format):
```
#42 (open) "Fix login bug"
  ├─> #45 (open) "Update auth flow"
  ├─> #38 (closed) "Security review"
  └─< #40 (open) "User report: can't login"

#45 (open) "Update auth flow"
  └─> owner/other-repo#12 (open) "API changes"
```

**Technical Approach**:
- Use GraphQL timeline API to get references
- Parse issue bodies for additional cross-references (regex)
- Build adjacency list/map for graph representation
- Recursive traversal for single-issue mode
- Format as tree structure with box-drawing characters

### 8. Output System (Phase 3)
**Purpose**: Provide flexible output options

**Requirements**:
- `--output <file>`: Write to file instead of stdout
- `--sort <field>`: Sort by created, updated, comments (default: created)
- `--order <dir>`: Sort order asc/desc (default: desc)
- `--format <format>`: Output format (text, json, dot) - default: text

**Technical Approach**:
- Use `encoding/json` for JSON output
- Custom formatters for text and DOT formats
- File I/O for `--output` flag

## Technical Stack

### Language & Runtime
- **Go**: 1.21+ (for modern features)
- **Single binary**: Cross-platform compilation

### Core Dependencies
- **CLI Framework**: `github.com/spf13/cobra` (command structure)
- **GitHub CLI SDK**: `github.com/cli/go-gh` (GitHub API client)
- **GraphQL**: Built into `go-gh`
- **Testing**: Standard `testing` package

### Development Tools
- **Linting**: `golangci-lint`
- **Formatting**: `gofmt` / `goimports`
- **Build**: Standard `go build`

## Project Structure
```
gh-issue-miner/
├── README.md
├── SPECIFICATION.md
├── go.mod
├── go.sum
├── main.go                 # Entry point
├── cmd/
│   ├── root.go            # Root command
│   ├── pulse.go           # Pulse command
│   └── graph.go           # Graph command
├── internal/
│   ├── api/
│   │   ├── client.go      # GitHub API client wrapper
│   │   ├── issues.go      # Issue fetching
│   │   └── queries.go     # GraphQL queries
│   ├── filter/
│   │   └── filter.go      # Filter logic
│   ├── analyzer/
│   │   ├── pulse.go       # Pulse metrics calculation
│   │   └── graph.go       # Graph building
│   ├── parser/
│   │   └── references.go  # Parse issue references
│   └── output/
│       ├── text.go        # Text formatting
│       ├── json.go        # JSON formatting
│       └── dot.go         # DOT formatting
└── internal/testutil/
    └── fixtures.go        # Test fixtures
```

## Implementation Phases

### Phase 1: Foundation (MVP)
**Goal**: GitHub CLI extension with `fetch` and `pulse` commands

- [x] Initialize Go module and project structure
- [x] Set up `cobra` CLI framework
- [x] Integrate `go-gh` for GitHub API
- [x] Implement repository detection (current repo or `--repo` flag)
- [x] Implement `fetch` command to list issues (supports `--repo` and `--limit`)
- [x] Implement `pulse` command to compute core metrics (supports `--repo` and `--limit`)
- [x] Test installation as `gh` extension

**Deliverable**: Users can run `gh issue-miner fetch` and `gh issue-miner pulse` against the current repository or a repository specified with `--repo`, with `--limit` available to bound results.

### Phase 2: Graph Command
**Goal**: Add issue relationship visualization

- [x] Implement `<url>` filter
- [x] Implement reference parser (regex) that detects full issue URLs, `owner/repo#123`, and short `#123` references
- [x] Populate `Issue.Body` from API responses so textual parsing can run over issue bodies
- [x] Add `ListIssueComments` helper to fetch issue comments for parsing
- [x] Add `graph` command (MVP): parse bodies, build adjacency map, print simple adjacency list
 - [x] Graph parses issue bodies and comments (single-issue and list modes)
- [ ] Add timeline API support and parse timeline items to discover timeline-originated references and explicit actions
- [ ] Extract and attach full edge metadata: actor (username), timestamp, action type (e.g., "referenced", "closed"), and source (`timeline` vs `parsed` vs `comment`)
- [ ] Comment-level attribution: capture comment id, author, and timestamp for parsed references and include that context on edges
- [ ] Resolve ambiguous short references (`#123`) when possible and surface unresolved/ambiguous short refs as warnings
- [ ] Implement recursive graph resolution with `--depth` and cycle detection; limit cross-repo expansion to avoid explosion
- [ ] Implement rate-limit and performance strategies: batching, exponential backoff, and limits for large repositories

### Phase 3: Enhanced Filtering & Pulse
**Goal**: Add comprehensive filtering and filter-aware pulse metrics

- [ ] Implement `--assignee` filter
- [ ] Implement `--author` filter
- [ ] Implement `--include-prs` filter
- [ ] Implement `--label` filter
- [ ] Implement `--state` filter
- [ ] Implement time range parser (relative and absolute dates)
- [ ] Implement `--created` filter (time range)
- [ ] Implement `--updated` filter (time range)
- [ ] Implement `--closed` filter (time range)
- [ ] Update pulse command to respect all filters
- [ ] Add filter summary to pulse output
- [ ] Add `--sort` and `--order` options
- [ ] Add JSON output format
- [ ] Add DOT output format for graphs
- [ ] Implement `--output` file option

**Deliverable**: Users can run filtered pulse queries like:
- `gh issue-miner pulse --label bug --created 30d` (bugs opened in last 30 days)
- `gh issue-miner pulse --author octocat --state closed --closed 90d` (issues by octocat closed in last 90 days)
- `gh issue-miner pulse --assignee alice --label enhancement` (enhancements assigned to alice)

### Phase 4: Polish & Distribution
**Goal**: Production-ready release

- [ ] Add comprehensive error handling
- [ ] Improve help text and documentation
- [ ] Add tests (unit and integration)
- [ ] Performance optimization
- [ ] Create installation instructions
- [ ] Tag release for `gh extension install`

## Installation & Usage

### Installation
```bash
gh extension install solvaholic/gh-issue-miner
```

### Basic Usage
#### Phase 1 (Pulse and Fetch Commands)
```bash
# Show pulse metrics for current repository
gh issue-miner pulse

# Show 50 issues from a specific repository
gh issue-miner fetch --repo cli/cli --limit 50
```

#### Phase 2 (Graph Command)
```bash
# Graph issue relationships
gh issue-miner graph --repo cli/cli --limit 50

# Graph specific issue
gh issue-miner graph https://github.com/cli/cli/issues/1234
```

#### Phase 3 (Enhanced Filters)
```bash
# Show pulse for bugs opened in last 30 days
gh issue-miner pulse --label bug --created 30d

# Show pulse for issues by specific author
gh issue-miner pulse --author octocat --state all

# Show pulse for closed issues in last 90 days assigned to alice
gh issue-miner pulse --assignee alice --state closed --closed 90d

# Graph bugs with relationships
gh issue-miner graph --label bug --closed 30d
```

### Authentication
No authentication needed! Uses `gh auth` automatically. Users must have GitHub CLI installed and authenticated:
```bash
gh auth login
```

## Repository Detection

When `--repo` is not specified:
1. Extract repository from `origin` remote URL
2. Check `GH_REPO` environment variable
3. Fall back to error if not detectable

## Error Handling

### Common Error Cases
- GitHub CLI not installed → "Please install GitHub CLI: https://cli.github.com"
- Not authenticated → "Please run: gh auth login"
- Invalid repository format → Show expected format: `owner/repo`
- API rate limit exceeded → Show wait time and current limit status
- Network errors → Retry with exponential backoff (3 attempts)
- Invalid filter combinations → Show error and suggest valid options

## Testing Strategy

### Unit Tests
- Test each command in isolation
- Mock GitHub API responses using interfaces
- Test filter logic with various inputs
- Test parser with different reference patterns
- Use table-driven tests (Go idiom)

### Integration Tests
- Test against real GitHub API (use test repository)
- Test as installed `gh` extension
- Test repository detection logic

### Test Coverage Target
- Minimum 70% code coverage
- 100% coverage for critical paths (API calls, parsing)

## Success Criteria

A successful v1.0 release includes:
1. ✅ Installable via `gh extension install`
2. ✅ Users can generate pulse metrics for any accessible repository
3. ✅ Pulse command respects all provided filters (labels, assignees, authors, time ranges)
4. ✅ Users can analyze specific issue subsets (e.g., bugs by author, closed in timeframe)
5. ✅ Users can visualize issue relationships in text format
6. ✅ Users can filter issues by state, label, assignee, author, created/closed dates
7. ✅ Clear documentation and help text with filter examples
8. ✅ No authentication code (uses `gh auth`)
9. ✅ Handles API errors gracefully
10. ✅ Single binary, works on macOS/Linux/Windows

## Future Enhancements (v2.0+)
- Additional filters (assignee, author, milestone, date ranges)
- Interactive mode for issue selection
- Configuration file support (`.issue-miner.yaml`)
- Multiple repository analysis (compare repos)
- Export to CSV
- Trend analysis over time
- Integration with GitHub Projects

## Distribution

### Release Process
1. Tag version: `git tag v1.0.0`
2. Push tag: `git push origin v1.0.0`
3. GitHub Actions builds binaries for all platforms
4. Users install/update via: `gh extension install solvaholic/gh-issue-miner`

### Supported Platforms
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)
