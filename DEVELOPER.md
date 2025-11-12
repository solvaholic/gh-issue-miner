DEVELOPER NOTES
===============

Quickstart for local development and testing of the `issue-miner` gh extension.

Local install (preferred during development)
-------------------------------------------
From the project root (where this README and `main.go` live) you can install the extension locally using `gh`:

```bash
# remove the released version if previously installed
gh extension remove gh-issue-miner

# build and install the extension from the current directory
go build -o gh-issue-miner .
gh extension install .

# verify extension is installed
gh extension list

# run the extension via gh
gh issue-miner --help
```

Notes:
- `gh extension install .` links to the extension in the current directory; `gh` will detect and use the executable found in the repo when running commands.
- When developing you can rebuild the binary and `gh` will use the updated version automatically.

Alternative: manual build + PATH
--------------------------------
If you prefer not to use `gh extension install .`, build the binary and put it on your `PATH`:

```bash
# build
go build -o gh-issue-miner .
# move to a PATH location
mkdir -p ~/.local/bin
mv gh-issue-miner ~/.local/bin/
chmod +x ~/.local/bin/gh-issue-miner
```

Testing
-------
- Unit tests: `go test ./...`
- Run the CLI directly (without install): `go run . fetch --repo owner/repo --limit 10`

Uninstalling
------------
Remove the locally installed extension:

```bash
gh extension remove gh-issue-miner
```

And replace it with the released version:

```bash
gh extension install solvaholic/gh-issue-miner
```

Troubleshooting
---------------
- If `gh extension install` fails, ensure you have the GitHub CLI (`gh`) installed and authenticated (`gh auth login`).
- If `gh` can't find the extension after install, make sure the executable is available in the root of your local repository.
