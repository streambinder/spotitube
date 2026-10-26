# Pango Constitution

**Status**: Draft. The repository owner ratifies this document.

The canonical standard for this repository is the unciae golden
standard (`streambinder/unciae`, file `AGENTS.md`). This constitution
restates the rules of that standard that bind this repository, in the
form that the Spec Kit workflow checks. When a rule here and the
canonical standard disagree, the canonical standard wins and this
document gets amended.

## Core Principles

### I. One Repository, One Deployable Unit

The repository holds one deployable unit: the `pango` command line
application. The root stays minimal: the `specs/` and `.specify/`
entries are the exception that section 25 of the standard
prescribes. (unciae §1, §2, §25)

### II. CI Is the Source of Truth

A change is done when the continuous integration pipeline is green.
Local checks mirror the pipeline; local checks never replace the
pipeline. Do not push on red. (unciae §1, §17, §18)

### III. One Linting System

super-linter is the linter. Its configuration files live only under
`.github/linters/`. No linter configuration sits at the repository
root, not even as a symlink; the workflow links or copies a
configuration to the root during the run only. Format code with the
canonical formatter of the language before every push: `gofumpt`
for Go, prettier for JSON, YAML, and Markdown. (unciae §3)

### IV. Tests and the Coverage Gate

Unit tests live next to the code that the tests cover. CI runs the
unit tests on every push. A failing test blocks the merge. CI
measures unit test coverage on every push, per package. The minimum
coverage is 90%. Coverage below 90% blocks the merge. Non-test code
never changes to satisfy a test; the test changes instead. Unit
tests stub every network dependency: a test never calls a live
provider. (unciae §15)

### V. Minimal Owned Surface

The repository owns its product logic and nothing else. For every
other functionality, first simplify the problem away, then delegate
to a specialized library with an established reputation. Pin exact
dependency versions where possible. (unciae §16, §16.1)

### VI. Commits and Pull Requests

Commits follow Conventional Commits: one logical change per commit,
signed, with no artificial-intelligence attribution trailers. Commit
messages and pull request descriptions never name a verification
subject; describe the class of the subject and the outcome instead.
The owner merges, except where the owner has delegated a merge in
writing for a stated class of change. (unciae §7, §19, §24)

### VII. Containers and Releases

A tag workflow never publishes `:latest`. The container image is
`ghcr.io/streambinder/pango` and targets linux/arm64. Release
binaries target every platform the project supports; the binary
matrix is a recorded exception to the arm64-only rule, because the
application runs on the machines of its users. Release builds
inject fallback credentials from CI secrets at link time. Source
code never contains a credential value. (unciae §5, §10, §11)

### VIII. Writing and Consistency

Technical writing follows ASD-STE100: short sentences, one term for
one concept, active voice. A change follows the conventions of the
file that the change touches. Do not reformat parts that the change
does not touch. (unciae §22, §23)

## Project Constraints

These constraints are decisions of the repository owner for this
project. Each constraint stands until the owner replaces it with a
recorded decision.

### Pipeline Shape

A synchronization is an assembly line of routines. Each routine
owns one stage and passes track objects through a dedicated queue:
index, authenticate, fetch, decide, collect, process, install, mix.
A stage keeps its failure policy local: a stage either drops the
track, degrades one enrichment, or aborts the run. The policy is
part of the stage contract and a change states its policy.

### Graceful Enrichment

Lyrics and artwork enrich a track; neither defines it. A failing
enrichment degrades to a warning and the track continues. Lyrics
collection queries every lyrics composer and picks the best result:
a synchronized text wins over a plain text, the longer text wins
otherwise. One failing composer never aborts the others; collection
fails only when every composer fails. The same rule binds the
asset providers of the decide stage: every provider is queried,
the best match wins, and one provider failure never aborts the
others.

### Honest Reporting

A report states only what the underlying value supports. A lookup
that fails stays distinguishable from a lookup with no result. A
missing value is reported from the value itself, never from a
derived projection of it: a projection that can be empty for a
present value, such as a first line, never decides whether the
value exists. A failure that a stage swallows stays visible in the
run output.

### Metadata Is the Index

The ID3 frames that the processor writes are the library index of
record. The indexer reads the Spotify identifier from a custom ID3
frame. Every reader of a stored track, including `show` and
`audit`, reads the same frames that the encoder writes, and a
display stays faithful to the stored value.

### Identity and Sources

Spotify is the source of truth for track identity and metadata.
An asset provider supplies a downloadable asset for a track. A
lyrics composer supplies a lyrics text for a track. Providers and
composers are interchangeable adapters behind their interfaces: a
new source joins by implementing the interface, and no stage
depends on one specific source.

### Library Surface

External tools import the packages of this module and pin the
module version. The exported surface of the `entity`, `lyrics`,
`spotify`, and `sys` packages is a compatibility contract: a
breaking change to that surface needs a recorded decision, and a
release tag carries the change to the consumers.

### Credentials

A credential reaches the application from an environment variable,
or from the fallback value that a release build injects at link
time. A credential never appears in source code, in a commit, or
in a pull request. Development builds carry no fallback value.

### Production Care

A production timer synchronizes the real library of the owner
every day, on a production host, through Imperium. The fix mode
re-downloads a track and replaces the local file. Before a change
to the collect, process, or install stages is declared ready, the
author verifies the change on a bounded selection: one track or a
small playlist, never a first run against the full library. A
verification reads the installed file back; a success line in the
run output is not verification.

### Specification Cycle

Every feature change and every domain change follows the
specification cycle: specify, plan, tasks, implement, verify,
converge. The specify step opens with an investigation of the
request as a hypothesis, on real data. The investigation record
states the evidence and the outcome. A specification is a draft
until the owner ratifies the specification. The artifacts live in
`specs/NNN-<short-name>/` at the repository root. A completed
feature directory is an immutable historical record. A bugfix and
a small change skip the cycle; the pull request body states the
rationale. A significant decision goes to the decision register
under `.specify/memory/decisions/`, in MADR format, one file per
decision. A decision is a draft until the owner ratifies it. An
accepted decision is reused; a superseded decision stays in the
register and names its successor.

## Working Agreements

This section restates the operative form of the principles, so the
repository stays self-contained for the points that the golden
standard states in full. A new session applies these agreements
without further sources.

### Branches and commits

- A feature branch carries the name `feat/<short-name>`. The name
  also covers fixes and chores.
- A commit follows Conventional Commits. The subject holds at most
  72 characters, uses the imperative, carries no terminal period,
  and contains none of the characters `()[]"'`/,:;` inside the
  subject text.
- A commit body holds one short paragraph. Each body line holds at
  most 100 characters. The body states why, not what, and ends with
  a period.
- One logical change makes one commit.
- A fix for a red feature branch amends the failing commit. The
  push uses force with an explicit lease that names the remote
  commit. Nobody amends or force-pushes the default branch.

### Text and formatting

- `gofumpt` formats Go code. Prettier formats every authored
  Markdown file. The author runs both formatters on each touched
  file before the push.
- Codespell checks every text file in CI. Authored text passes
  codespell before the push.
- Spec Kit manages the files under `.specify/`, except
  `.specify/memory/`. Nobody edits a managed file by hand. The lint
  pipeline excludes the managed subtrees through targeted rules in
  the filter setting and in the linter configurations.

### Verification

- A change passes `gofumpt`, `golangci-lint`, the full test suite
  with the race detector, and the coverage gate, locally and in CI.
- A change that touches a synchronization stage is verified on a
  bounded real selection before the change is declared ready, and
  the pull request states the evidence in generic terms.

## Governance

The owner ratifies this constitution and every amendment. The agent
drafts amendments and records the ratified text. Every pull request
review checks compliance with this constitution. A rule that proves
wrong is amended through a recorded decision; the rule is never
silently ignored.

**Version**: 1.0.0 | **Ratified**: pending | **Last Amended**: 2026-10-26
