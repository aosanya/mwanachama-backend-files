package models

type Folder struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Audience  string   `json:"audience"`
	GroupIDs  []string `json:"group_ids,omitempty"`
	CreatedBy string   `json:"created_by,omitempty"`
	CreatedAt string   `json:"created_at"`
}

type FolderFilter struct {
	Audience string `query:"audience"`
}

type File struct {
	ID             string `json:"id"`
	FolderID       string `json:"folder_id"`
	Name           string `json:"name"`
	CurrentVersion int    `json:"current_version"`
	CreatedBy      string `json:"created_by,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type Version struct {
	ID          string `json:"id"`
	FileID      string `json:"file_id"`
	Number      int    `json:"number"`
	Name        string `json:"name,omitempty"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
	StorageKey  string `json:"-"`
	UploadedBy  string `json:"uploaded_by,omitempty"`
	UploadedAt  string `json:"uploaded_at"`
	Content     []byte `json:"content,omitempty" spec:"-"`
}
