# mwanachama-backend-files

Holds the files an agency keeps: documents in folders, photo and file
answers, a report's PDF, the documents on a record. Every file is a record
with its versions; the bytes sit behind a store the host chooses.

Text files with a history of their own belong in
[mwanachama-backend-git](https://github.com/aosanya/mwanachama-backend-git);
everything else lands here.

A standalone Go library with no server of its own. `mwanachama-wakala-api`
mounts it per agency under `/instances/{id}/files` and decides who may read
what.

- Requirements: [documentation/1. requirements/requirements.md](documentation/1.%20requirements/requirements.md)
- Board: [documentation/3. implementation/todo.md](documentation/3.%20implementation/todo.md)
