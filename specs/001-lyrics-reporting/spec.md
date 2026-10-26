# Feature Specification: Truthful Lyrics Reporting

**Feature Branch**: `feat/lyrics-reporting`

**Created**: 2026-10-26

**Status**: Draft. The owner ordered this fix on 2026-10-26, after
the investigation below, and ordered the SDD adoption first. The
owner ratifies this text at the review of the pull request.

**Input**: Report by the owner, 2026-10-26: on one track, the
synchronization printed the lyrics as not found, a companion tool
printed the lyrics for the same track minutes later, and the `show`
command still reported the lyrics frame as unset after both runs.

## Hypothesis Investigation

**Hypothesis**: A synchronization reports the lyrics of a track as
missing while the lyrics sources hold them, and the failure is in
the search or in the persistence of the lyrics frame.

**Evidence** (verified on 2026-10-26, on live sources, on the
machine of the owner, and by reproduction):

1. The lyrics cache file for the track existed after the run, with
   the full lyrics text, and the synchronization itself had
   written the file during the run. `lyrics.Search` writes the
   cache only on a hit, so the search had found the text.
2. The cached text begins with an empty line. The page parser of one
   lyrics composer emits a line break before the first text node,
   so every text from that composer starts with an empty first
   line.
3. Both reporting sites reduce the value to its first line through
   `sys.FirstLine`: the collect log line in `cmd/sync.go` and the
   Lyrics row in `cmd/show.go`. The first line of the text is
   empty, and both sites fall back on an empty excerpt: a "not
   found" log line in the synchronization, an unset row in `show`.
4. The companion tool consumes the same `lyrics.Search` result and
   excerpts the whole text, so the tool displayed the lyrics. The
   tool also saved the same lyrics frame again, which is why the
   installed file kept its size across its runs while its
   timestamps moved: the persistence had worked in both tools.
5. Reproduction: a file that carries a lyrics frame whose text
   begins with an empty line reads the full text back through the
   ID3 layer, and `show` reports the frame as unset.

**Assumed cause check**: The cause is not a source outage: the
sources held the text and the run cached it. The cause is not a
persistence failure: a byte scan of the installed file finds the
lyrics sections, and the frame reads back in full. The cause is
not a search failure. The cause is an unnormalized lyrics text at
the lyrics boundary, plus a lossy projection that both displays
use to decide whether the value exists. The constitution now
states the rule in the section Honest Reporting.

**Outcome**: Specify a smaller correction than the report implied.
No change to the search and no change to the persistence. The
lyrics boundary returns text without leading empty lines, and both
displays read the first non-empty line.

## User Scenarios and Testing

### User Story 1 - A synchronization reports a lyrics hit truthfully (Priority: P1)

The owner runs a synchronization. The collect stage finds the
lyrics of a track. The run output shows an excerpt of the lyrics.

**Why this priority**: The run output is the primary signal of the
tool. A false negative hides every real success and sends the
owner to debug a healthy stage.

**Independent Test**: Run a synchronization against a scripted
lyrics source that answers a text with a leading empty line. The
printed lyrics line carries the first content line, not the
not-found fallback.

**Acceptance Scenarios**:

1. **Given** a lyrics source that returns a text starting with
   empty lines, **When** the collect stage processes the track,
   **Then** the run output shows the first non-empty line of the
   text.
2. **Given** a lyrics source that returns no text, **When** the
   collect stage processes the track, **Then** the run output
   shows the not-found fallback.

### User Story 2 - Show reports a stored lyrics frame truthfully (Priority: P1)

The owner runs `show` on an installed track. The track carries a
lyrics frame. The Lyrics row shows an excerpt of the stored text.

**Why this priority**: Stored frames from runs before the fix keep
the leading empty line. Normalizing new results alone cannot repair
the reporting for the existing library.

**Independent Test**: Write a fixture file whose lyrics frame
starts with empty lines and run `show`. The Lyrics row carries the
first non-empty line, not the unset fallback.

**Acceptance Scenarios**:

1. **Given** a stored lyrics frame that starts with empty lines,
   **When** the owner runs `show`, **Then** the Lyrics row shows
   the first non-empty line of the frame.
2. **Given** a stored track without a lyrics frame, **When** the
   owner runs `show`, **Then** the Lyrics row shows the unset
   fallback.

### User Story 3 - Lyrics consumers receive a clean text (Priority: P2)

A tool that imports the `lyrics` package calls `Search` or `Get`.
The returned text starts at its first content line, and the cache
file on disk holds the same text.

**Why this priority**: Consumers excerpt the returned text and
write it into files. A shared normalization at the boundary
repairs every consumer at once.

**Independent Test**: Stub a composer that returns a text starting
with empty lines. `Search` and `Get` return the text without the
leading empty lines, and the cache file holds the normalized text.

**Acceptance Scenarios**:

1. **Given** a composer that returns a text starting with empty
   lines, **When** a consumer calls `Search`, **Then** the result
   and the cache file start at the first content line.
2. **Given** a cache file written by an earlier version, with
   leading empty lines, **When** a consumer calls `Search`,
   **Then** the result starts at the first content line.
3. **Given** a cache file that holds only empty lines, **When** a
   consumer calls `Search`, **Then** the cache counts as a miss and
   the composers run.

## Requirements

### Functional Requirements

- **FR-001**: `lyrics.Search` and `lyrics.Get` return a lyrics
  text with no leading empty lines.
- **FR-002**: The lyrics cache stores the normalized text, and a
  cache read normalizes a legacy entry. A cache entry that holds
  only empty lines counts as a miss.
- **FR-003**: The collect log line and the `show` Lyrics row
  excerpt the first non-empty line of the value. The fallback text
  appears only for an actually empty value.
- **FR-004**: The lyrics choice rules stay unchanged: a
  synchronized text wins over a plain text, the longer text wins
  otherwise, and collection fails only when every composer fails.
- **FR-005**: The stored lyrics frame format stays unchanged, and
  no other stage changes behavior.

### Key Entities

- **Lyrics text**: the plain or synchronized text that a composer
  returns for a track, held in the cache, in the track object, and
  in the lyrics frame of the installed file.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Unit tests prove that a composer result with leading
  empty lines yields a normalized result and a normalized cache
  file from `Search`, and a normalized result from `Get`.
- **SC-002**: A command test proves that `show` prints the first
  content line for a stored frame with leading empty lines.
- **SC-003**: A command test proves that a full synchronization
  whose lyrics source answers with leading empty lines prints the
  first content line, not the not-found fallback.
- **SC-004**: The full test suite passes with the per-package
  coverage gate at 90 percent, locally and in CI.
