package files

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-files/models"
)

type (
	Folder       = models.Folder
	FolderFilter = models.FolderFilter
	File         = models.File
	Version      = models.Version
)

type FileManager interface {
	CreateFolder(ctx context.Context, in Folder) (Folder, error)
	ListFolders(ctx context.Context, filter FolderFilter) ([]Folder, error)

	CreateFile(ctx context.Context, in File) (File, error)
	GetFile(ctx context.Context, fileID string) (File, error)

	AddVersion(ctx context.Context, in Version) (Version, error)
	ListVersions(ctx context.Context, fileID string) ([]Version, error)
}

type fileManager struct {
	db    *gorm.DB
	st    *store
	bytes ByteStore
	now   func() string
}

func NewFileManager(db *gorm.DB, s *spec.Spec, bytes ByteStore) (FileManager, error) {
	if db == nil {
		return nil, fmt.Errorf("NewFileManager: db must not be nil")
	}
	if bytes == nil {
		return nil, fmt.Errorf("NewFileManager: a byte store is required")
	}
	st, err := newStore(db, s, map[string]any{
		roleFolder:  models.Folder{},
		roleFile:    models.File{},
		roleVersion: models.Version{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewFileManager: %w", err)
	}
	return &fileManager{db: db, st: st, bytes: bytes, now: nowStamp}, nil
}

func nowStamp() string { return time.Now().UTC().Format(specstore.TimeLayout) }
