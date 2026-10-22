package files

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-files/models"
)

func (m *fileManager) CreateFolder(ctx context.Context, in Folder) (Folder, error) {
	if err := m.checks(roleFolder, in, ErrInvalidFolder); err != nil {
		return Folder{}, err
	}
	if in.Audience == models.AudienceGroups && len(in.GroupIDs) == 0 {
		return Folder{}, fmt.Errorf("%w: a folder for groups names at least one group", ErrInvalidFolder)
	}
	if in.Audience != models.AudienceGroups {
		in.GroupIDs = nil
	}
	in.ID = newID()
	in.CreatedAt = m.now()
	if err := m.st.Insert(ctx, roleFolder, in); err != nil {
		return Folder{}, fmt.Errorf("CreateFolder: %w", err)
	}
	return in, nil
}

func (m *fileManager) ListFolders(ctx context.Context, filter FolderFilter) ([]Folder, error) {
	q := m.st.Query(ctx, roleFolder)
	if filter.Audience != "" {
		q = q.Where("audience = ?", filter.Audience)
	}
	out, err := specstore.List[Folder](m.st, q.Order("created_at desc"), roleFolder)
	if err != nil {
		return nil, fmt.Errorf("ListFolders: %w", err)
	}
	return out, nil
}

func (m *fileManager) getFolder(ctx context.Context, folderID string) (Folder, error) {
	var out Folder
	q := m.st.Query(ctx, roleFolder).Where("id = ?", folderID)
	if err := m.st.Take(q, roleFolder, &out, ErrFolderNotFound); err != nil {
		return Folder{}, err
	}
	return out, nil
}
