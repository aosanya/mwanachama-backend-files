# Files settings (FL3)

Member app decision 318 (owner, 2026-10-22): photos are shrunk on the phone
before they are sent, and how far, how many per question and the largest file
are this module's settings, starting at about 1600 px, 5 per question and
20 MB.

## Declared the S64 way

The three settings sit in `files.blueprint.json`'s `settings` block, which
`mwanachama-backend-shared/spec` validates at load and copies onto every
loaded `Spec` (see shared's
[declared-domains.md](../../../mwanachama-backend-shared/documentation/2.%20design/declared-domains.md#a-module-declares-its-settings)).
A domain spec cannot carry them.

| Setting | Type | Default | Who acts on it |
| --- | --- | --- | --- |
| `photo_longest_side` | int | 1600 (pixels) | the consumer that shrinks a photo (the member app) |
| `photos_per_question` | int | 5 | the consumers that collect photos (forms F86, the member app) |
| `largest_file_bytes` | int | 20971520 (20 MiB) | this module, on every upload |

This module never resizes an image and never counts photos: the first two
are exposed so the consumers read one value rather than each hard-coding
their own.

## Resolved in layers

`FileManager.Settings(ctx)` answers `Settings`: the three values as typed
fields plus `Effective`, each setting with the layer that set it, from
`spec.ResolveAll` — an owner's override, else the agency definition's value,
else the declared default. A layer whose value no longer fits the shape is
passed over.

The module stores neither layer; S64 has not built per-agency storage yet.
A host that holds them supplies them with `UseSettings(SettingsLayers)`, a
function answering the definition values and the overrides for the request's
context. With none supplied, every setting is its declared default. An error
from the layers is returned, not swallowed into the default, so an upload is
never checked against a limit the agency did not set.

`GET /settings` (action `files.settings.read`, tool derived by `dispatch`)
serves the same answer, gated like every other address.

## Checked on upload

`AddVersion` resolves the settings and refuses content larger than
`largest_file_bytes` with `ErrFileTooLarge`, mapped to **413** in
`files.operations.json`, before any byte reaches the byte store. A file
exactly at the limit is accepted.

## Not settled here

- "20 MB" is taken as 20 MiB (20 × 1024 × 1024 bytes). If the owner meant
  20 000 000 bytes the default changes; nothing else does.
- A host's own request-body limit may refuse a large upload before it
  reaches this module, with its own status.
