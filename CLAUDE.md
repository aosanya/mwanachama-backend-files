# CLAUDE.md

Guidance for Claude Code working in this repository.

## Project: mwanachama-backend-files

Holds an agency's files and their versions: documents in folders, photo and
file answers to forms, a report's PDF, the documents on a record. Module path
`github.com/aosanya/mwanachama-backend-files`. Created 2026-10-22 (member app
decision 312).

**A declared domain from the first commit.** The objects come from
`files.blueprint.json` and the route table from `files.operations.json`;
storage is `mwanachama-backend-shared/specstore` and the routes are built by
`mwanachama-backend-shared/dispatch.Table`. There are no row structs and no
`AutoMigrate`. `mwanachama-backend-catalog` is the reference;
`mwanachama-backend-sources` is the nearest template for the file layout.

**Standalone library, no server of its own**, and **no auth model**: every
address is gated and the mounting host decides who may read a folder, a
record's documents or an answer's photo.

## The rules that matter most

- **Records here, bytes elsewhere.** The module owns the record of a file and
  its versions (name, content type, size, checksum, who uploaded, when). The
  bytes go to a byte store the host passes in. Where production keeps them is
  on hold (decision 313); a local-disk store serves development and tests.
- **Text with a history belongs in git.** This module never diffs or merges.
- **This module names no domain.** A folder's audience is data the host reads;
  the words "council", "matter" or "tender" never appear here.
- **The uploader comes from the caller**, never the body.

## Conventions

- Four-phase `documentation/` layout — see
  [documentation/README.md](documentation/README.md).
- `go test ./...` (sqlite) verifies a change.
- **Adding a field means editing `files.blueprint.json` and the Go type in
  `models/` together.**
- An address is an entry in `files.operations.json`.

## Code comments

Write code with no comments. Not one-liners above a function, not section
banners, not doc comments on exported symbols, not "why" notes next to a
tricky line. A name, a type, or a smaller function carries it instead.

Anything that genuinely needs explaining goes in this repo's `documentation/`
folder, under the phase it belongs to (`1. requirements`, `2. design`,
`3. implementation`, `4. qa`) — never inline.

**Why:** inline prose drifts out of sync with the code, duplicates what
`documentation/` already owns, and buries the explanation where nobody
looking for it will search.

**How to apply:**

- New code ships without comments. If a line seems to need one, rename or
  split until it doesn't.
- Touching code that already has comments: strip the ones in the code you are
  changing. Do not sweep untouched files unless asked.
- If the reasoning matters, add or update the matching `documentation/` page
  in the same change and leave nothing behind in the source.
- Machine-read directives are not comments and stay: build tags, `//go:embed`,
  `//go:generate`, linter pragmas (`//nolint`, `// eslint-disable-next-line`,
  `// ignore:`), license headers, codegen "do not edit" banners, and generated
  files as a whole.
- Commit messages, PR descriptions, and test names carry the narration that
  used to go in comments.

This rule is repeated verbatim in every mwanachama repo's `CLAUDE.md` so that
it reaches sessions that do not load this machine's user-level config —
scheduled cloud routines, other machines, and other agent harnesses.
