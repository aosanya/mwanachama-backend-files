# Reading a file

FL2, from member app decisions 314, 316 and 317. `read.go`.

## The host decides who may read

This module has no auth model and names no domain, so it never knows what a
folder's audience means, who a record's people are, or who may see a form
answer. The host passes a `ReadCheck` to `NewReader`:

    func(ctx, caller string, folder Folder, file File) (Standing, error)

It answers one of three standings:

| Standing | Meaning |
| --- | --- |
| `StandingNone` | may not read this file |
| `StandingReader` | may read the latest version (a member, decision 316) |
| `StandingStaff` | may read every version in the history (decision 316) |

The check runs **before** a link is issued. A refused caller, a caller with no
id, a missing file and a version the caller may not name all answer
`ErrFileNotFound`, so a probe cannot tell a file that exists from one that
does not. An error the check itself returns is passed through unchanged.

## Members get the latest, staff every version

`Reader.Versions(ctx, caller, fileID)` lists newest first: staff get the whole
history, a reader only the current version. `Reader.Link(ctx, caller, fileID,
number)` takes `number` 0 for the latest; a reader naming any earlier version
is answered not-found, staff may name any.

## A short-lived link

`Link` returns a `DownloadLink{token, expires_at, version}`. The token is
`base64url(file_id|number|expires_unix) . base64url(HMAC-SHA256(key, claim))`.
The key is the host's, at least 32 bytes; the lifetime is the host's too.
`Reader.Now` is settable so tests move the clock.

`Reader.Open(ctx, token)` redeems it: a token whose signature does not match is
`ErrLinkInvalid`, one at or past its expiry is `ErrLinkExpired`, and a valid
one returns the version and its bytes from the byte store. The link carries
the grant: `Open` does not run the check again, so a link stays good until it
expires even if the caller's standing changes within its lifetime. That is why
the lifetime should be short.

## Not built here

- No address in `files.operations.json` yet. `Link` and `Open` sit on
  `Reader`, not on `FileManager`, because the caller's check and the key are
  the host's; the host mounts a route that calls them, and redeeming a link is
  necessarily an unauthenticated address (the token is the credential).
- No revocation list: a link cannot be withdrawn before it expires.
