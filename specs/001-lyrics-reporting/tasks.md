# Tasks: Truthful Lyrics Reporting

**Input**: Design documents from `/specs/001-lyrics-reporting/`

**Prerequisites**: plan.md (required), spec.md (required)

**Tests**: The constitution requires unit tests next to the code
and a 90 percent coverage gate. The task list includes the tests.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to
- Include exact file paths in descriptions

## Phase 1: Artifacts

**Purpose**: Close the specify and plan steps of cycle 001.

- [x] T001 Record the hypothesis investigation and the
      requirements in `specs/001-lyrics-reporting/spec.md`
- [x] T002 Record the normalization point and the projection
      decision in `specs/001-lyrics-reporting/plan.md`

## Phase 2: Lyrics Normalization (Story 3)

**Purpose**: The boundary returns a text that starts at its first
content line. Story 1 depends on this phase through `Search`.

- [x] T003 [US3] Add the normalization helper to
      `lyrics/composer.go` and apply the helper to the merged
      result in `Search`, to the cache entry on read, and to the
      result in `Get`
- [x] T004 [US3] Cover the normalization in
      `lyrics/composer_test.go`: a composer result with leading
      empty lines, the cache file content, a legacy cache entry,
      an empty-only cache entry, and a `Get` result

## Phase 3: Display Projection (Stories 1 and 2)

**Purpose**: Both displays excerpt the first non-empty line, so
the fallback fires only for an empty value.

- [x] T005 [P] [US1] Add `sys.FirstContentLine` next to
      `sys.FirstLine` in `sys/string.go`, with cases in
      `sys/string_test.go`
- [x] T006 [US1] Switch the collect log line in `cmd/sync.go` to
      the new projection
- [x] T007 [US2] Switch the Lyrics row in `cmd/show.go` to the
      new projection
- [x] T008 [US2] Cover the stored-frame case in
      `cmd/show_test.go`: a fixture whose lyrics frame starts
      with empty lines prints the first content line
- [x] T009 [US1] Cover the run output in `cmd/sync_test.go`: the
      web script in `cmd/harness_test.go` gains a lyrics answer
      with leading empty lines, and the printed lyrics line
      carries the first content line

## Phase 4: Verification

**Purpose**: Prove the cycle against the constitution gates.

- [x] T010 Run gofumpt, golangci-lint, the full test suite with
      the race detector, and the coverage gate locally
- [x] T011 Open the pull request stacked on the onboarding branch
      and reach a green pipeline

## Phase 5: Converge

**Purpose**: Close the cycle as a historical record.

- [x] T012 Mark the task outcomes and the success criteria in
      this file at the end of the implementation
