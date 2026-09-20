# agent-skills-validator

[![Release](https://github.com/coolapso/agent-skills-validator/actions/workflows/release.yaml/badge.svg?branch=main)](https://github.com/coolapso/agent-skills-validator/actions/workflows/release.yaml)
![GitHub Tag](https://img.shields.io/github/v/tag/coolapso/agent-skills-validator?logo=semver&label=semver&labelColor=gray&color=green)
[![Container image](https://img.shields.io/badge/ghcr.io-coolapso%2Fagent--skills--validator-blue?logo=docker)](https://github.com/coolapso/agent-skills-validator/pkgs/container/agent-skills-validator)
[![Go Report Card](https://goreportcard.com/badge/github.com/coolapso/agent-skills-validator)](https://goreportcard.com/report/github.com/coolapso/agent-skills-validator)
![GitHub Sponsors](https://img.shields.io/github/sponsors/coolapso?style=flat&logo=githubsponsors)

An independent Go CLI and GitHub Action that validate skill directories against the public
[Agent Skills specification](https://agentskills.io/specification).

Website: <https://agent-skills-validator.coolapso.sh/>

It checks the portable `SKILL.md` contract and nothing else: no Markdown quality opinions,
no host-specific fields, no repository conventions, no network access.

---

## Features

- **Every specification rule** for `SKILL.md` frontmatter, each with a stable rule ID (`AS001`–`AS010`)
- **Actionable diagnostics** with file path, line number and severity
- **Text or JSON output** for humans and CI
- **Strict mode** that promotes specification recommendations to errors
- **Unicode aware**: lengths are counted in code points, CRLF and BOM files are accepted, malformed UTF-8 is reported clearly
- **Cross-platform** binaries for Linux, macOS and Windows
- **GitHub Action** that downloads a checksum-verified release binary; no Go, Python or Node on the runner

---

## Usage

```
agent-skills-validator validate [flags] <skill-directory>...

Flags:
      --fail-on-warnings   exit with code 1 when warnings are present
      --format string      output format: text or json (default "text")
  -h, --help               help for validate
      --strict             treat specification recommendations as errors instead of warnings
```

```sh
# one skill
agent-skills-validator validate ./skills/pdf-processing

# every skill in a directory, machine readable
agent-skills-validator validate --format json skills/*/

# recommendations are errors and any warning fails the run
agent-skills-validator validate --strict --fail-on-warnings ./my-skill
```

Example output:

```
skills/pdf-processing: valid
skills/data/SKILL.md:2: error AS004: name "data-analysis" must match the skill directory name "data"
skills/data/SKILL.md:501: warning AS010: SKILL.md has 512 lines; the specification recommends keeping it under 500 lines and moving detail into referenced files

2 skills validated, 1 valid, 1 invalid (1 error, 1 warning)
```

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | No errors. Warnings may be present unless `--fail-on-warnings` is set. |
| `1` | Validation errors, or warnings promoted to failures. |
| `2` | CLI misuse or an input path that does not exist or cannot be read. |

### Rules

Run `agent-skills-validator rules` to print the table below together with the
specification revision the release was built against.

| ID | Severity | Rule |
| --- | --- | --- |
| `AS001` | Error | Target is a directory containing `SKILL.md`. |
| `AS002` | Error | `SKILL.md` is valid UTF-8, begins with YAML frontmatter and has a closing delimiter. |
| `AS003` | Error | `name` is a string of 1-64 Unicode lowercase alphanumeric characters or single hyphens; it does not start or end with a hyphen. |
| `AS004` | Error | `name` matches the parent directory name. |
| `AS005` | Error | `description` is a non-empty string of at most 1024 characters. |
| `AS006` | Error | When present, `license` is a string. |
| `AS007` | Error | When present, `compatibility` is a 1-500 character string. |
| `AS008` | Error | When present, `metadata` is a mapping of string keys to string values. |
| `AS009` | Error | When present, `allowed-tools` is a string. |
| `AS010` | Warning | `SKILL.md` exceeds the specification's 500-line recommendation. |

Unknown frontmatter keys and extra files or directories are allowed by the specification and
never produce diagnostics.

### JSON output

`--format json` emits one document suitable for CI parsing. Field meanings are stable
within a major version; new fields may be added.

```json
{
  "version": 1,
  "rulesVersion": 1,
  "skills": [
    {
      "path": "skills/example",
      "valid": true,
      "diagnostics": []
    },
    {
      "path": "skills/data",
      "valid": false,
      "diagnostics": [
        {
          "rule": "AS004",
          "severity": "error",
          "path": "skills/data/SKILL.md",
          "line": 2,
          "message": "name \"data-analysis\" must match the skill directory name \"data\""
        }
      ]
    }
  ]
}
```

`valid` is `true` when the skill has no errors. Warnings do not affect `valid`; use
`--strict` to turn them into errors or `--fail-on-warnings` to fail the exit code.

---

## GitHub Action

```yaml
jobs:
  validate-skills:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: coolapso/agent-skills-validator@v1
        with:
          path: skills/terraform-skill
          strict: true
```

| Input | Default | Meaning |
| --- | --- | --- |
| `path` | `.` | Skill directory to validate. |
| `version` | Pinned per action release | CLI release to download. Use `latest` for the newest release. |
| `strict` | `false` | Pass `--strict`. |
| `fail-on-warnings` | `false` | Pass `--fail-on-warnings`. |
| `format` | `text` | CLI output format. With `json` the report is also exposed as action outputs. |

| Output | Meaning |
| --- | --- |
| `exit-code` | The CLI exit code. |
| `json` | The JSON report when `format: json`. |
| `json-file` | Path to a file holding the JSON report when `format: json`. |

The action downloads the release binary for the runner platform from GitHub Releases, verifies it
against the published `checksums.txt`, and runs it. Linux, macOS and Windows runners are supported.

Validate several skills with a matrix:

```yaml
strategy:
  matrix:
    skill: [skills/pdf-processing, skills/data-analysis]
steps:
  - uses: actions/checkout@v5
  - uses: coolapso/agent-skills-validator@v1
    with:
      path: ${{ matrix.skill }}
```

Consume the JSON report:

```yaml
- uses: coolapso/agent-skills-validator@v1
  id: skills
  continue-on-error: true
  with:
    path: skills/pdf-processing
    format: json
- run: echo '${{ steps.skills.outputs.json }}' | jq '.skills[].diagnostics'
```

---

## Installation

### Docker

Images for `linux/amd64` and `linux/arm64` are published to the GitHub Container Registry.
Mount the directory holding your skills on `/data` (the image's working directory) and pass the
skill paths relative to it:

```sh
docker run --rm -v "$PWD:/data:ro" ghcr.io/coolapso/agent-skills-validator:latest validate skills/pdf-processing

# pin a version, machine-readable output
docker run --rm -v "$PWD:/data:ro" ghcr.io/coolapso/agent-skills-validator:1.0.0 validate --format json skills/*/
```

The image is built from `scratch`, runs as a non-root user, and contains only the static binary.

### Install script (Linux, macOS)

Downloads the latest release, verifies the checksum and installs to `/usr/local/bin`:

```sh
curl -fsSL https://agent-skills-validator.coolapso.sh/install.sh | bash
```

Pin a version or change the destination:

```sh
curl -fsSL https://agent-skills-validator.coolapso.sh/install.sh | VERSION=v1.0.0 INSTALL_DIR=~/.local/bin bash
```

Uninstall with `curl -fsSL https://agent-skills-validator.coolapso.sh/uninstall.sh | bash`.

### Go install

```sh
go install github.com/coolapso/agent-skills-validator@latest
```

### Arch Linux (AUR)

```sh
yay -S agent-skills-validator-bin
```

### Debian, Ubuntu, Fedora, RHEL

Download the `.deb` or `.rpm` package from the
[releases page](https://github.com/coolapso/agent-skills-validator/releases).

### Manual install

Download the archive for your platform from the
[releases page](https://github.com/coolapso/agent-skills-validator/releases), verify it against
`checksums.txt`, extract it and put the binary on your `PATH`. The install script above does
exactly this and is also what the GitHub Action runs.

### Build from source

```sh
task build      # or: go build -o agent-skills-validator .
```

---

## Development

The project uses [Task](https://taskfile.dev). Run `task` to list everything.

```sh
task tools:install   # semrel, GoReleaser, golangci-lint, semrel plugins
task check           # formatting, vet, lint, tests, fixture smoke test
task test:cover      # tests with coverage
task snapshot        # build all release artifacts into ./dist without publishing
task container:smoke # build the container image and run it against the fixtures
task site:serve      # preview the website with live reload (needs go-live-server)
```

The website under `site/` is plain HTML and is published to Cloudflare Pages at
<https://agent-skills-validator.coolapso.sh> by `task site:deploy` or by the
`Deploy site to Cloudflare Pages` workflow on every push to `main` that touches it.

Every rule has fixtures under `internal/validator/testdata`. Add a fixture and a table entry in
`internal/validator/validator_test.go` whenever a rule changes.

### Releasing

Releases follow [Conventional Commits](https://www.conventionalcommits.org) and can be run
entirely from a local machine or from the `Release` workflow; both call the same tasks.

```sh
task release:dry-run   # preview the next version and notes
task release           # checks, semrel tag + GitHub release, GoReleaser artifacts, move v1 tag
```

[semrel](https://semrel.io) computes the version, pins the action's default CLI version in
`action.yaml`, commits the changelog, tags and creates the GitHub release.
[GoReleaser](https://goreleaser.com) then builds the binaries, archives, checksums, `.deb`/`.rpm`
packages and the AUR package and attaches them to that release. Finally the multi-arch container
image is pushed to the GitHub Container Registry. A token is taken from `GITHUB_TOKEN` or
`gh auth token`; `AUR_KEY` and the Discord webhook variables are optional.

### Specification revision

Each release documents the Agent Skills specification revision it implements. The current
rules follow the specification as published on 2026-09-20; run
`agent-skills-validator rules` to see the revision baked into your binary.

---

## Contributions

Improvements and suggestions are always welcome!
Check open issues, or open a new Issue or Pull Request.

If you like this project and want to support or contribute in another way, you can [:heart: Sponsor Me](https://github.com/sponsors/coolapso) or:

<a href="https://www.buymeacoffee.com/coolapso" target="_blank">
  <img src="https://cdn.buymeacoffee.com/buttons/default-yellow.png" alt="Buy Me A Coffee" style="height: 51px !important;width: 217px !important;" />
</a>
