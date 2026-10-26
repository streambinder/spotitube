# Implementation Plan: Truthful Lyrics Reporting

**Branch**: `feat/lyrics-reporting` | **Date**: 2026-10-26 |
**Spec**: `specs/001-lyrics-reporting/spec.md`

**Input**: Feature specification from
`/specs/001-lyrics-reporting/spec.md`

## Summary

The lyrics boundary learns one normalization, and the two lyrics
displays learn one projection. A helper in `lyrics/composer.go`
strips leading whitespace from a lyrics text. `Search` applies
the helper to the merged result before the cache write and the
return, to a cache entry on read, and `Get` applies the helper to
its result. A new `sys.FirstContentLine` returns the first line
whose trimmed content is not empty, and the collect log line in
`cmd/sync.go` and the Lyrics row in `cmd/show.go` excerpt through
that projection instead of `sys.FirstLine`. Nothing else changes:
the composer merge rules, the failure accounting, and the frame
format stay as they are.

## Technical Context

**Language/Version**: Go, version from `go.mod`

**Primary Dependencies**: the standard library only; no new
dependency

**Storage**: the lyrics cache file under the XDG cache directory,
one file per track

**Testing**: table and stub tests with the established suite;
command tests run against scripted transports and a fake tool PATH

**Constraints**: per-package coverage at 90 percent; unit tests
stub every network dependency; the exported surface of the
`lyrics` package stays compatible

## Constitution Check

- Graceful Enrichment: the change keeps the degradation contract.
  Collection still fails only when every composer fails.
- Honest Reporting: the change implements the rule directly. The
  fallback of each display now depends on the value, not on a
  projection that can be empty for a present value.
- Library Surface: `Search` and `Get` keep their signatures.
  Consumers receive a cleaner text at their next pin bump; no
  breaking change.
- Specification Cycle: this document and the task list complete
  the plan and tasks steps of cycle 001.

## Decisions

- **Normalization point**: one helper at the lyrics boundary, not
  one fix per composer. A new composer can emit the same leading
  empty lines tomorrow; the boundary rule covers every source,
  the legacy cache, and both entry points in one place.
- **Normalization form**: strip the leading whitespace run. A
  synchronized text starts with a timestamp bracket and carries no
  leading whitespace, so the rule leaves synchronized texts
  untouched. Lyrics lines carry no significant leading whitespace.
- **Legacy cache**: normalize on read instead of migrating files.
  A cache entry that normalizes to empty held no content, so the
  read falls through to the composers.
- **Display projection**: add `sys.FirstContentLine` next to
  `sys.FirstLine`. `FirstLine` keeps its contract for its other
  semantic; the two lyrics displays switch to the new projection.
- **No change in the companion tools**: the tools import the
  `lyrics` package and inherit the normalization when they bump
  the module pin. Their own excerpts already read the whole text.

## Project Structure

### Documentation (this feature)

```text
specs/001-lyrics-reporting/
├── spec.md
├── plan.md
└── tasks.md
```

### Source Code (repository root)

```text
lyrics/composer.go        # normalization helper, applied in Search and Get
lyrics/composer_test.go   # normalization, cache, and legacy cache cases
sys/string.go             # FirstContentLine next to FirstLine
sys/string_test.go        # projection cases
cmd/sync.go               # collect log line uses the new projection
cmd/show.go               # Lyrics row uses the new projection
cmd/show_test.go          # stored frame with leading empty lines
cmd/sync_test.go          # end-to-end lyrics line through a scripted source
cmd/harness_test.go       # the web script gains a leading-empty lyrics answer
```
