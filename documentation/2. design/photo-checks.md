# Photo checks on upload

FL4, member app decision 320: members may take a photo or pick one; a question
may require "taken now, with time and place". Otherwise no place is kept and a
photo's hidden location data is removed.

`AddVersion` runs `checkPhoto` (`photocheck.go`) before anything is stored.

## The content type comes from the bytes

`net/http.DetectContentType` reads the first bytes of the upload, and its
answer is the version's `content_type`. Whatever the body declared, and
whatever the name says, is overwritten. A version named `report.pdf` whose
bytes are a JPEG is stored as `image/jpeg`.

## The upload says whether time and place are required

`Version.RequireTimeAndPlace` (`require_time_and_place` on the wire) is an
option of the upload, not a stored column (`spec:"-"`). This module does not
know the question; the caller that does — forms, through the host — sets it.

**Required.** The photo must be a JPEG whose EXIF carries
`DateTimeOriginal` (tag `0x9003` in the Exif IFD) and both `GPSLatitude`
(`0x0002`) and `GPSLongitude` (`0x0004`) in the GPS IFD, each with its three
rationals. Anything else — no EXIF, no time, no place, or a PNG or WebP — is
refused with `ErrTimeAndPlaceRequired` (422) and nothing is stored. A photo
that passes is stored byte for byte as uploaded, so the capture time and place
stay in the file. They are not copied into columns.

**Not required.** Location data is removed before the bytes reach the byte
store:

| Type | What is removed |
| --- | --- |
| JPEG | every APP1 segment (`0xFFE1`) — EXIF and XMP alike, since XMP can carry a place too |
| PNG | every `eXIf` chunk; other chunks keep their own CRCs, so none is recomputed |
| WebP | every `EXIF` chunk; the RIFF size is rewritten and the EXIF flag in `VP8X` cleared |

The checksum and size are taken after stripping, so they describe the bytes
held. Every other type is stored unchanged.

A JPEG, PNG or WebP whose structure cannot be walked is refused with
`ErrUnreadablePhoto` (400) rather than stored with its metadata intact.

## Known limits

- Removing APP1 also removes the EXIF orientation tag, so a phone photo that
  relied on it may display rotated. Clients normally rotate before upload.
- PNG `iTXt`/`tEXt` chunks holding XMP and WebP `XMP ` chunks are not removed.
- HEIC is not sniffed by `DetectContentType` and is stored unchanged.
- A required time-and-place photo is accepted on presence of the tags; their
  values are not checked against the time of upload.
