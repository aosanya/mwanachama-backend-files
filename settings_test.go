package files_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	files "github.com/aosanya/mwanachama-backend-files"
)

func TestTheSettingsStartAtTheDecidedDefaults(t *testing.T) {
	fm := newTestManager(t)
	s, err := fm.Settings(context.Background())
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if s.PhotoLongestSide != 1600 || s.PhotosPerQuestion != 5 || s.LargestFileBytes != 20*1024*1024 {
		t.Fatalf("settings = %+v", s)
	}
	for _, e := range s.Effective {
		if e.Layer != spec.LayerModule {
			t.Fatalf("%s came from %q, want the module's default", e.Name, e.Layer)
		}
	}
}

func TestAnOverrideOutranksTheDefinitionAndAValueOffShapeIsPassedOver(t *testing.T) {
	fm := newTestManager(t)
	fm.UseSettings(func(context.Context) (map[string]any, map[string]spec.Override, error) {
		return map[string]any{"photos_per_question": 3, "photo_longest_side": "big"},
			map[string]spec.Override{"photos_per_question": {Value: 8, By: "actor-9", At: "2026-10-22T10:00:00Z"}},
			nil
	})
	s, err := fm.Settings(context.Background())
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if s.PhotosPerQuestion != 8 || s.PhotoLongestSide != 1600 {
		t.Fatalf("settings = %+v", s)
	}
	for _, e := range s.Effective {
		if e.Name == files.SettingPhotosPerQuestion && (e.Layer != spec.LayerOverride || e.By != "actor-9") {
			t.Fatalf("photos per question = %+v", e)
		}
	}
}

func TestAnUploadPastTheLargestFileIsRefused(t *testing.T) {
	ctx := context.Background()
	fm := newTestManager(t)
	fm.UseSettings(func(context.Context) (map[string]any, map[string]spec.Override, error) {
		return map[string]any{"largest_file_bytes": 4}, nil, nil
	})
	folder, _ := fm.CreateFolder(ctx, files.Folder{Name: "Papers", Audience: "staff"})
	f, _ := fm.CreateFile(ctx, files.File{FolderID: folder.ID, Name: "a.txt"})
	if _, err := fm.AddVersion(ctx, files.Version{FileID: f.ID, ContentType: "text/plain", Content: []byte("12345"), UploadedBy: "actor-1"}); !errors.Is(err, files.ErrFileTooLarge) {
		t.Fatalf("err = %v, want ErrFileTooLarge", err)
	}
	if _, err := fm.AddVersion(ctx, files.Version{FileID: f.ID, ContentType: "text/plain", Content: []byte("1234"), UploadedBy: "actor-1"}); err != nil {
		t.Fatalf("an upload at the largest file was refused: %v", err)
	}
	got, _ := fm.GetFile(ctx, f.ID)
	if got.CurrentVersion != 1 {
		t.Fatalf("current version = %d, want 1", got.CurrentVersion)
	}
}

func TestTheLayersFailingFailsTheUpload(t *testing.T) {
	ctx := context.Background()
	fm := newTestManager(t)
	boom := errors.New("layers unavailable")
	fm.UseSettings(func(context.Context) (map[string]any, map[string]spec.Override, error) { return nil, nil, boom })
	folder, _ := fm.CreateFolder(ctx, files.Folder{Name: "Papers", Audience: "staff"})
	f, _ := fm.CreateFile(ctx, files.File{FolderID: folder.ID, Name: "a.txt"})
	if _, err := fm.AddVersion(ctx, files.Version{FileID: f.ID, ContentType: "text/plain", Content: []byte("x"), UploadedBy: "actor-1"}); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the layers' error", err)
	}
}
