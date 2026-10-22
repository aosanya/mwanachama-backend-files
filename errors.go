package files

import "errors"

var (
	ErrFolderNotFound = errors.New("folder not found")
	ErrFileNotFound   = errors.New("file not found")
	ErrInvalidFolder  = errors.New("invalid folder")
	ErrInvalidFile    = errors.New("invalid file")
	ErrInvalidVersion = errors.New("invalid version")
	ErrNoUploader     = errors.New("an upload needs a caller to stand behind it")
	ErrBytesNotFound  = errors.New("bytes not found")
	ErrInvalidKey     = errors.New("invalid storage key")
	ErrFileTooLarge   = errors.New("file is larger than the largest file setting")
)
