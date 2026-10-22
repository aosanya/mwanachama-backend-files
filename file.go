package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func (m *fileManager) CreateFile(ctx context.Context, in File) (File, error) {
	if err := m.checks(roleFile, in, ErrInvalidFile); err != nil {
		return File{}, err
	}
	if _, err := m.getFolder(ctx, in.FolderID); err != nil {
		return File{}, err
	}
	in.ID = newID()
	in.CurrentVersion = 0
	in.CreatedAt = m.now()
	in.UpdatedAt = in.CreatedAt
	if err := m.st.Insert(ctx, roleFile, in); err != nil {
		return File{}, fmt.Errorf("CreateFile: %w", err)
	}
	return in, nil
}

func (m *fileManager) GetFile(ctx context.Context, fileID string) (File, error) {
	var out File
	q := m.st.Query(ctx, roleFile).Where("id = ?", fileID)
	if err := m.st.Take(q, roleFile, &out, ErrFileNotFound); err != nil {
		return File{}, err
	}
	return out, nil
}

func (m *fileManager) AddVersion(ctx context.Context, in Version) (Version, error) {
	if in.UploadedBy == "" {
		return Version{}, ErrNoUploader
	}
	if len(in.Content) == 0 {
		return Version{}, fmt.Errorf("%w: content is required", ErrInvalidVersion)
	}
	if err := m.fitsLargestFile(ctx, len(in.Content)); err != nil {
		return Version{}, err
	}
	if err := m.checks(roleVersion, in, ErrInvalidVersion); err != nil {
		return Version{}, err
	}
	held, err := m.GetFile(ctx, in.FileID)
	if err != nil {
		return Version{}, err
	}

	sum := sha256.Sum256(in.Content)
	in.ID = newID()
	in.StorageKey = newID()
	in.Size = int64(len(in.Content))
	in.Checksum = hex.EncodeToString(sum[:])
	in.UploadedAt = m.now()

	if err := m.bytes.Put(ctx, in.StorageKey, bytes.NewReader(in.Content)); err != nil {
		return Version{}, fmt.Errorf("AddVersion: %w", err)
	}

	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		in.Number = held.CurrentVersion + 1
		row, err := encode(m.st, roleVersion, in)
		if err != nil {
			return err
		}
		if err := tx.Table(m.st.Table(roleVersion)).Create(row).Error; err != nil {
			return err
		}
		res := tx.Table(m.st.Table(roleFile)).
			Where("id = ? AND current_version = ?", held.ID, held.CurrentVersion).
			Updates(map[string]any{"current_version": in.Number, "updated_at": in.UploadedAt})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("%w: the file moved on while uploading", ErrInvalidVersion)
		}
		return nil
	})
	if err != nil {
		_ = m.bytes.Delete(ctx, in.StorageKey)
		return Version{}, fmt.Errorf("AddVersion: %w", err)
	}
	in.Content = nil
	return in, nil
}

func (m *fileManager) ListVersions(ctx context.Context, fileID string) ([]Version, error) {
	if _, err := m.GetFile(ctx, fileID); err != nil {
		return nil, err
	}
	q := m.st.Query(ctx, roleVersion).Where("file_id = ?", fileID).Order("number desc")
	out, err := specstore.List[Version](m.st, q, roleVersion)
	if err != nil {
		return nil, fmt.Errorf("ListVersions: %w", err)
	}
	return out, nil
}

func encode(st *store, role string, v any) (map[string]any, error) {
	return specstore.Encode(st.Object(role), v)
}
