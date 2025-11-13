# Design Notes for gh-issue-miner

Purpose
-------
This file collects implementation decisions, trade-offs, and rationale that are important for maintainers and contributors but are intentionally separated from the product-level `SPECIFICATION.md` (which describes WHAT the tool should do).

Spec vs Design (guidance)
-------------------------
- `SPECIFICATION.md` (the spec) states product requirements, user-facing behavior, and acceptance criteria: "what" the application must provide (commands, filters, formats, success criteria). Keep it focused on user-visible behavior and high-level architecture (e.g., we use GraphQL/REST).
- `DESIGN.md` (this file) records non-obvious implementation choices, constraints, and rationale: "why" and "how" things were implemented. It explains trade-offs, testing seams, performance implications, and places where implementation differs from an ideal future architecture. Use this file for decisions that influence future changes.

Non-obvious decisions and rationale
----------------------------------
1. Server vs client filtering
   - Decision: Push filters to GitHub where the REST API supports them exactly (state, exact labels, sort/direction, and `since` for updated-start). Wildcard label prefixes, time upper-bounds, and other complex combinations are enforced client-side.
   - Rationale: The GitHub REST API (via `labels=` and list issues endpoints) doesn't support some semantics we want (for example OR across labels or arbitrary date-range upper bounds). Doing a conservative server pushdown minimizes data transfer where possible while preserving correctness by performing additional client-side filtering when necessary.
   - Implication: Some queries fetch extra candidates and perform client-side filtering; therefore `--limit` is enforced after client-side filters and we fetch extra candidates to avoid truncation surprises (see Candidate-fetch strategy).

2. Label wildcard expansion
   - Decision: Trailing-* label specs (e.g., `priority/*`) are expanded by listing repository labels. Exact matches produced by expansion are pushed to the server; prefixes that don't match any repo label remain as client-side fallback filters.
   - Rationale: Converting prefixes to exact labels lets the REST API perform efficient server-side filtering when possible while falling back to local filtering when necessary.
   - Implication: The expansion step requires an extra repo label listing API call for queries that include wildcards.

3. Labels AND vs OR semantics
   - Decision: When multiple labels are sent to the REST `labels=` query parameter, GitHub treats them as an AND (issues must have all labels). We use that behavior for server pushdown of exact labels.
   - Rationale: It's the REST API's semantics; changing it would require Search/GraphQL queries or multiple server requests.
   - Implication / Note for users: `--label a,b` behaves as AND when labels are pushed server-side. If OR semantics are required, we must implement a Search/GraphQL-based path (planned for Phase 4).

4. Time-range pushdown
   - Decision: If `--updated` has a left/start bound, we pass it to the server via the `since` parameter. Upper bounds (end) and created/closed full-range semantics remain client-side.
   - Rationale: The `since` parameter is supported and reduces transferred data; but the REST list API doesn't support arbitrary end bounds reliably in all contexts.
   - Implication: Server results may include items outside the requested end bound; those are filtered locally. This affects performance but keeps correctness.

5. Candidate-fetch strategy and `--limit`
   - Decision: When client-side filtering may drop results (wildcards, time-end checks), the CLI fetches extra candidates (default multiplier 3×, capped at 2000) and applies client-side filters, then trims to `--limit`.
   - Rationale: Ensure users get the expected number of results after client-side filtering while avoiding infinite fetch loops.
   - Implication: Heavily filtered queries may use more API requests and bandwidth.

6. Sort+Limit interaction
   - Decision: `--sort`/`--direction` are applied server-side (when supported) and thus affect which items are returned before client-side filtering and trimming.
   - Rationale: Server ordering is necessary to limit the dataset meaningfully (e.g., top 10 by most comments).
   - Implication: Sorting is part of selection semantics; documenting this helps users reason about results.

7. Test seams and helper APIs
   - Decision: `internal/api.ListIssues` is exposed via a package-level variable `ListIssuesFunc` so tests can override it. `cmd.FetchIssues` centralizes list + client-side filtering logic for easier testing.
   - Rationale: Makes unit tests deterministic and avoids hitting the real API during unit tests.
   - Implication: Keep this seam stable; refactors should maintain the test seam or update tests accordingly.

8. Concurrency and API load control
   - Decision: Parallel API operations (comments and timeline fetches) are bounded by semaphores/worker pools and results cached to avoid repeated requests.
   - Rationale: Avoid API bursts and rate-limit pressure; improve efficiency by reusing timeline data when multiple nodes refer to the same issue.

9. Date/time semantics
   - Decision: Dates parse as UTC day boundaries (start = 00:00 UTC); relative forms like `7d` mean last 7×24h. Ranges `left..right` support open ends and relative specifications on either side.
   - Rationale: Provides predictable behavior across systems and aligns with how we compare timestamps from the API.

10. Positional-URL exclusivity
    - Decision: When a positional issue URL is provided, it is exclusive with selection filters. The CLI checks and returns an error if filters are used with a positional URL.
    - Rationale: Single-issue mode is conceptually different from list-mode; mixing them is ambiguous for output and execution flow.

Where to document these decisions
---------------------------------
- Short pointers / usage notes should appear in `README.md` near examples (labels/time/limit behavior) so users read them quickly.
- `SPECIFICATION.md` should reference `DESIGN.md` and stay focused on the product-level contract (what the tool does). For example, a brief sentence under the Filtering System: "Implementation notes and trade-offs are recorded in `DESIGN.md`."
- Longer rationale, trade-offs, and developer notes (this file) should live in `DESIGN.md` so maintainers can find the background tech-debt and migration roadmap.

Recommended small edits (I can apply them if you want)
------------------------------------------------------
- Add a one-line pointer in `SPECIFICATION.md` under Filtering System: "See `DESIGN.md` for implementation decisions and trade-offs affecting filtering semantics."
- Add a short section in `README.md` labeled "Behavior notes" summarizing labels/time/limit implications with links to `DESIGN.md`.
- Add `DEVELOPER.md` notes showing how to use `ListIssuesFunc` and `FetchIssues` for test mocks.

Maintenance guidance
--------------------
- When changing server pushdown behavior (e.g., moving to GraphQL Search and pushing OR semantics or full date-range), update `DESIGN.md` with the migration plan and the backward-compatibility considerations.
- If the test seam (`ListIssuesFunc`) is removed or renamed, update tests and document the new mocking strategy in `DEVELOPER.md`.

Done / Next actions
-------------------
- I created `DESIGN.md` in the repository with the above content and added a pointer in `SPECIFICATION.md`. If you want, I can also:
  - Add the brief `README.md` note linking to design (recommended).
  - Create `DEVELOPER.md` with testing examples (mocking `ListIssuesFunc`).

Which of those (README pointer, DEVELOPER.md) should I add next? If you prefer, I can apply them now. (I will mark the todo item completed once you confirm.)