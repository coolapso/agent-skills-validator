# AI Agents Guide for `agent-skills-validator`

This file is the entry point for AI coding agents working on this repository. Read it before
changing anything. It is both the product specification and the guide to how the code implements
it. User-facing behaviour (flags, exit codes, rule table, JSON contract, action inputs) is
documented in `README.md`; keep the two in sync.

## Project overview

`agent-skills-validator` is a Go CLI (and a composite GitHub Action) that validates skill
directories against the public [Agent Skills specification](https://agentskills.io/specification).
It validates the portable `SKILL.md` frontmatter contract only. It deliberately does **not**
validate Markdown quality, agent behaviour, host-specific fields, repository conventions,
external URLs or reference-link existence, and it never touches the network.

## Project structure

- `main.go` — entry point; calls `cmd.Execute()` and exits with its code.
- `cmd/` — cobra command tree. `root.go` builds the root command and maps errors to exit codes,
  `validate.go` implements `validate`, `rules.go` prints the rule table, `cmd_test.go` tests exit
  codes through `cmd.Run` with in-memory streams. `cmd.Version` is set by GoReleaser via `-ldflags`.
- `internal/validator/` — the validation engine.
  - `rules.go` — rule IDs, severities, limits, `Rules` table, `SpecRevision` and `RulesVersion`.
  - `frontmatter.go` — UTF-8 check, BOM/CRLF handling, delimiter detection and YAML decoding into
    `yaml.Node` trees (scalar types preserved).
  - `validator.go` — `Validate`, `ValidateAll`, per-field checks and `NameProblem`.
  - `result.go` — `Diagnostic`, `Result`, `Report` (the JSON contract).
  - `format.go` — text and JSON writers.
  - `testdata/` — fixtures: `valid/`, `invalid/`, `warn/`, and `golden/report.json`.
- `action.yaml` — the composite GitHub Action. It runs `site/install.sh` with `INSTALL_DIR` set to
  the runner's temp directory, then executes the binary.
- `site/` — the static website served by Cloudflare Pages at
  <https://agent-skills-validator.coolapso.sh/> (project `agent-skills-validator`, deployed with
  wrangler by `task site:deploy` or the `deploypages.yaml` workflow; never GitHub Pages):
  `index.html` (plain HTML with Tailwind from
  the CDN, no build step), `install.sh` (the single installer: curl-pipe for users and the download
  step of the action; verifies `checksums.txt`, supports `VERSION`, `INSTALL_DIR`, `GH_TOKEN`) and
  `uninstall.sh`. Keep the rule table and flags on the page in sync with the CLI and README.
  Preview it with `task site:serve`, which uses the user's own
  [go-live-server](https://github.com/coolapso/go-live-server) (`live-server` binary) for live
  reload; never substitute another dev server.
- `Dockerfile` + `.dockerignore` — multi-stage image: Go builder cross-compiling for
  `TARGETOS`/`TARGETARCH`, `scratch` runtime, non-root user, `WORKDIR /data` for bind mounts.
  Built and pushed by the `container:*` tasks to `ghcr.io/coolapso/agent-skills-validator` and
  Docker Hub `coolapso/agent-skills-validator`, tagged `latest` and the release version.
- `Taskfile.yml` — every build, test and release command. CI calls these tasks; do not duplicate
  logic in workflows.
- `.semrel.yaml`, `.semrel.lock`, `.goreleaser.yaml` — release tooling configuration.
- `.github/workflows/` — `test.yaml` (lint, tests, fixtures on Linux/macOS/Windows),
  `action-test.yaml` (runs the action against fixtures on three OSes once a release exists),
  `release.yaml` (runs `task release:ci`), `deploypages.yaml` (publishes `site/` to Cloudflare Pages).

## Design decisions you must preserve

- **Rules map 1:1 to the specification.** Every diagnostic has a stable rule ID (`AS001`–`AS010`),
  a severity, a path, a line when known, and an actionable message. Do not add checks that go
  beyond the public specification; if the specification gains a requirement, add it here first,
  then implement. Unknown frontmatter keys and extra files never produce diagnostics.
- **No regular expressions for frontmatter parsing.** Frontmatter is split by line and decoded with
  `go.yaml.in/yaml/v3` into `yaml.Node` so scalar types are preserved: `version: 1.0` is a float,
  `version: "1.0"` is a string. Type checks use `node.Tag` (`!!str`, `!!int`, ...).
- **Lengths are Unicode code points** (`utf8.RuneCountInString`), never bytes.
- **`name` character class** is `unicode.IsLower` or `unicode.IsDigit` or `-`; uppercase gets a
  dedicated message. AS004 (directory match) is only evaluated when AS003 passes, to avoid noise.
- **Exit codes:** `0` no errors, `1` errors or promoted warnings, `2` misuse or an input path that
  cannot be stat'ed or read. A directory that exists but has no `SKILL.md` is an AS001 error
  (exit 1), not misuse. `validator.InputError` is the type that signals exit 2.
- **`--strict`** promotes warnings to errors inside the validator (so JSON `valid` reflects it);
  **`--fail-on-warnings`** only changes the exit code in `cmd`.
- **Paths in output** are slash-normalised with `filepath.ToSlash` so JSON is identical across
  platforms.
- **JSON contract** is `version: 1`. Fields may be added; existing meanings must not change within
  a major CLI version. `TestJSONOutputStability` compares against `testdata/golden/report.json`;
  regenerate it only for intentional changes with `task test:golden:update`.
- **yaml.v3 error line numbers** are not consistently 1-based, so the AS002 line for YAML syntax
  errors is best effort. Tests only assert it falls inside the frontmatter block.
- **Fixtures are binary-exact.** `.gitattributes` disables line-ending conversion under
  `internal/validator/testdata` so the CRLF fixture stays CRLF on every platform. The malformed
  UTF-8 case is generated in a temp directory by the test rather than committed.
- **The action never compiles Go.** It downloads the GoReleaser archive named
  `agent-skills-validator_<version>_<os>_<arch>.<tar.gz|zip>` and verifies it against
  `checksums.txt`. Changing archive or checksum names in `.goreleaser.yaml` requires changing
  `site/install.sh` too. The script must keep working under `shell: bash` on Windows runners
  (Git Bash): no `install`, `unzip` may be missing (7z and PowerShell fallbacks exist).
- **Release runs never mutate the repository.** No generated commits, no file rewrites, no
  changelog. semrel tags the commit that is already on the remote and creates the GitHub release;
  GoReleaser and the container push only publish artifacts. `release:gh` refuses to run on a
  dirty tree or when local and remote branches differ.
- **The action's `matrix` input defaults to `true` and must keep today's behaviour.** With
  `matrix: true`, `path` is a single skill directory passed to the CLI untouched, which is what
  every caller using `path: ${{ matrix.skill }}` relies on; more than one line is exit 2 with a
  hint. With `matrix: false`, `path` is newline-delimited and lines holding `*`, `?` or `[` are
  glob-expanded by bash in the validate step (the CLI never globs); literal lines still go to the
  CLI so a missing path stays a CLI exit 2. A composite action cannot create jobs, so the
  job-per-skill fan-out lives in the caller's workflow. Changing the default is a breaking change.
- **The action follows its own ref for the CLI version.** With `version` empty, the install step
  uses `github.action_ref`: an exact tag runs that CLI release, a major tag like `v1` runs the
  newest `v1.x` release (resolved by `site/install.sh` through the releases API), anything else
  runs `latest`. Action and CLI are released together, so the refs line up.

## Adding or changing a rule

1. Confirm the rule exists in the public specification, update the design decisions above if
   needed, and bump `SpecRevision` in `rules.go` when the specification revision changed.
2. Add the check in `validator.go` and the row in `Rules`. Bump `RulesVersion` when a rule is
   added, removed or changes meaning.
3. Add at least one fixture under `testdata/invalid/<rule>-<case>/SKILL.md` (and a `valid/` one
   if the rule has an edge that must pass), then add the expectation to the table in
   `TestInvalidFixtures`. The test fails if a fixture has no table entry.
4. Update the rule table in `README.md`.

## Non-goals

These are deliberate and must not be "fixed":

- Validating Markdown quality, agent behaviour, host-specific frontmatter fields, repository
  conventions, external URLs, or whether referenced files exist. Those are outside the portable
  format contract.
- Reimplementing `skills-ref` or becoming an alternative Agent Skills specification.
- Validating a repository's `AGENTS.md`, release flow, license policy, or host-specific packaging.
- Automatically fixing skill files.
- Requiring a package manager or runtime other than the released Go binary.
- Network access of any kind during validation.

## Deferred ideas

- **One-level local reference checks** (does `references/REFERENCE.md` mentioned in `SKILL.md`
  exist?). Only consider this once the Agent Skills specification defines the exact parsing
  semantics for file references, and keep it disabled by default.
## Secrets

No GitHub repository secrets are used. Credentials live in the Infisical project `cicd`
(EU cloud, environment `ci`): `ghcr`, `dockerhub`, `aur`, `cloudflare_api_token`, `cloudflare_account_id`,
and optionally `discord_webhook_id` / `discord_webhook_token`.

- Locally, tasks fall back to `task secrets:get -- <key>` (the `infisical` CLI, user login) when
  the matching environment variable is unset; GHCR falls back further to the `gh` token.
- In CI, `Infisical/secrets-action` authenticates with GitHub OIDC (job needs `id-token: write`)
  and exports each key as an environment variable with the same lowercase name, e.g.
  `${{ env.aur }}`. The machine identity's OIDC subject must match this repository's token. This
  repo uses GitHub's immutable subject format with numeric IDs:
  `repo:coolapso@14358086/agent-skills-validator@1378153445:ref:refs/heads/main`. Glob `*` does
  not cross `/`, so the binding needs a trailing `**`. Fetch the prefix with
  `gh api repos/coolapso/agent-skills-validator/actions/oidc/customization/sub`.
- Never print secret values in task output or logs. Listing key names is fine.

## Release process

Releases are Conventional-Commit driven and runnable from a laptop or CI with the same tasks:

- `task release:dry-run` shows the next version.
- `task release` runs `check`, then `release:gh` (`semrel release` creates the tag and GitHub
  release; `commit_changelog: false`, release notes are the changelog), then `release:artifacts`
  (`goreleaser release --clean` attaches binaries, checksums, deb/rpm and AUR), then
  `release:major-tag` moves `v1` to the new tag, then `container:push` publishes the multi-arch
  image (`docker buildx`, needs QEMU for arm64 locally).
- **Floating major tags must stay out of semrel's way.** semrel finds the last release with
  `git describe --tags --abbrev=0`, and a floating tag (`v0`, `v1`) on the same commit as the
  release tag wins that lookup, so semrel would report `v0` as the current version. Every task
  that calls semrel depends on `release:drop-floating-tags`, which deletes those tags from the
  local clone first (`git fetch --tags` restores them). `release:major-tag` takes the release tag
  from `.semrel-release.json` (or the tags on `HEAD`), never from `git describe`, and tags the
  peeled commit (`^{commit}`) so the floating tag is always a lightweight tag on a commit, never a
  copy of semrel's annotated tag object. A copied tag object carries the release tag's name and
  makes `git describe` print `v0.1.0-2-g<sha>`, which semrel resolves to `HEAD` and then reports
  "No commits since last release".
- Commit types that release: `feat` (minor), `fix`/`ref`/`build` (patch). Use `chore`, `docs`,
  `test`, `ci` for changes that must not release.
- `.semrel.lock` pins `@semrel/provider-github`; it must be a version compatible with the pinned
  semrel core (0.5.3 with semrel v0.26.2, the same lock as convcommitlint and the terraform
  modules). An older plugin fails with "This binary is a plugin. These are not meant to be
  executed directly." Update the lock by copying it from a working repo or with
  `semrel plugin update`.

## Mandatory checks before you finish

Run these and fix everything they report:

```sh
task check        # gofmt, go vet, golangci-lint, go test -race, fixture smoke test
shellcheck site/install.sh site/uninstall.sh
goreleaser check
task container:smoke   # when you touched the Dockerfile
```

`site/install.sh` can be tested without a release: build a binary, pack it as
`agent-skills-validator_<ver>_linux_amd64.tar.gz` next to a `checksums.txt`, serve that directory
with `python3 -m http.server`, and run the script with `VERSION=<ver> BASE_URL=http://127.0.0.1:<port>`.

If you changed `action.yaml` or the workflows, keep them valid YAML and remember that composite
action steps run with `shell: bash` on all three runner operating systems, including Windows
(Git Bash), so avoid GNU-only flags.

## Commit messages

Use Conventional Commits with the types above. Scope by area when useful, e.g.
`feat(validator): ...`, `fix(action): ...`, `build(release): ...`.
