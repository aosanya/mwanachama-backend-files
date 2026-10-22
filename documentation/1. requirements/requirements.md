# Requirements

## Problem

Agencies hold files that are not text with a history: a photo on a pothole
report, a site inspector's evidence, a CV, a report's PDF, council papers, a
client's deliverable. Until now the only place bytes could land was
`mwanachama-backend-git`, which keeps each file inside the database as base64
and checks nobody's right to read it.

## Decisions

Settled by the owner in the member app's
[requirements.md](https://github.com/aosanya/mwanachama-frontend-member/blob/master/documentation/1.%20requirements/requirements.md),
2026-10-22.

| # | Rule |
|---|---|
| 312 | Files get their own module; git keeps text files only. Every file is a record with its versions. A report's PDF and its versions live here. |
| 313 | Where the bytes are kept in production is on hold; the owner leans to storage inside the Kubernetes cluster. Until then a swappable store, local disk for development and tests. |
| 314 | Documents sit in folders, each choosing who reads it: public, members, chosen groups, or staff. A record's documents are read by that record's people. |
| 316 | Staff upload, replace (the old version kept) and remove (kept in the history); members see only the latest version. |
| 317 | A photo or file answer is seen by whoever may see that answer or item; on a public board item only after staff approve it for publishing. |
| 318 | Photos are shrunk on the phone; how far, how many per question and the largest file are this module's settings, starting at about 1600 px, 5 per question and 20 MB. |
| 320 | A question may require a photo "taken now, with time and place"; otherwise no place is kept and a photo's location data is removed. |

## Open questions

| # | Question |
|---|---|
| Q19 (member app) | Can a report also be written as text in the studio? If so, that text lives in git. |
