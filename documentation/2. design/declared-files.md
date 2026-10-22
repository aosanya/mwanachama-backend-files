# The declared files module (FL1)

What the scaffold holds and why it is shaped this way. Settled by member app
decisions 312–314 and 316.

## Three objects

`files.blueprint.json` declares them; a domain spec only names them.

| Role | Holds |
|---|---|
| `folder` | a name and its **audience** — `public`, `members`, `groups` (with `group_ids`) or `staff`. The audience is data the host reads to decide who may read; this module never decides it. |
| `file` | the record: its folder, its name and `current_version`. |
| `version` | one upload: number, name, content type, size, SHA-256 checksum, `storage_key`, `uploaded_by`, `uploaded_at`. Never edited. |

`files.wakala.json` is the shipped domain (`SpecFor`). `spec/examples/` holds
a council's papers and a firm's deliverables, which share no noun, so the
module's neutrality is exercised. `cmd/ddl <spec> [postgres|sqlite]` prints
the DDL.

## Records here, bytes elsewhere

`AddVersion` takes the bytes in `Version.Content` (base64 on the wire; tagged
`spec:"-"`, never a column), counts them, hashes them, writes them to the
`ByteStore` under a freshly minted key, then records the version and moves
the file's `current_version` in one transaction. The move is guarded on the
number it read, so two uploads racing cannot both claim the same number (the
unique `(file_id, number)` index is the second guard). If the record fails,
the bytes just written are deleted. `storage_key` is `json:"-"`: it never
reaches a response.

`ByteStore` is `Put`/`Get`/`Delete` by key. `DiskStore` is the development and
test store; it accepts only keys in `[A-Za-z0-9][A-Za-z0-9._-]{0,127}`, so a
key cannot leave its directory, and writes through a temporary file renamed
into place. Where production keeps the bytes is on hold (decision 313).

## The caller

`created_by` and `uploaded_by` are bound `from: caller`; a body naming
someone else is overwritten. An upload with no caller is `ErrNoUploader`,
answered 401. The folder and file in an address outrank the body.

## Gating

`AnonymousActions` is empty and `TestNoRouteIsAnonymous` pins it: every
address arrives gated, and the host's authorizer, reading a folder's
audience, decides who passes.

## Not built here

Removing a file (kept in history), the member-facing "latest version only"
read, reading the bytes back over HTTP, the settings of decision 318 and the
location rule of decision 320 are later rows.
