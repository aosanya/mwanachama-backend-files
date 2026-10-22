# Task board — pending

| ID | Task | Status | Depends on |
|---|---|---|---|
| FL1 | 🧭 **Scaffold the module in the declared-domain shape** (member app decisions 312–314, 316, 2026-10-22): blueprint with a file, its versions (name, content type, size, checksum, uploaded by from the caller, when) and a folder with its audience (public, members, groups, staff) held as data; operations and `dispatch.Table` routes; specstore; `cmd/ddl`; the guard tests (domain-agnostic, every address gated); bytes behind a `ByteStore` interface with a local-disk store. Unblocks FL2–FL4, wakala-api WK200. | 🚀 2026-10-22 13:23Z · c2a95f | — |
| FL2 | 🧭 **Reading a file** (member app decisions 314, 316, 317): the host's check (a folder's audience, a record's people, an answer's visibility) runs before a short-lived download link is issued; members get the latest version, staff every version. | ⏸️ | FL1 |
| FL3 | 🧭 **Files settings** (member app decision 318): the longest side a photo is shrunk to (1600 px), photos per question (5), the largest file (20 MB), declared through shared S64 and checked on upload. | ⏸️ | FL1 |
| FL4 | 🧭 **Photo checks on upload** (member app decision 320): the content type read from the bytes, not the name; location data removed unless the question requires time and place, and then the capture time and place required and kept. | ⏸️ | FL1 |
| FL5 | 🧭 **A byte store on the cluster** (member app decision 313): production bytes on Kubernetes storage. On hold by the owner. | ⏸️ | owner |
