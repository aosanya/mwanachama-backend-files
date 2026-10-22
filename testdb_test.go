package files_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	files "github.com/aosanya/mwanachama-backend-files"
)

func newTestManager(t *testing.T) files.FileManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := files.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := files.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	disk, err := files.NewDiskStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewDiskStore: %v", err)
	}
	fm, err := files.NewFileManager(db, s, disk)
	if err != nil {
		t.Fatalf("NewFileManager: %v", err)
	}
	return fm
}

func TestAFileKeepsEveryVersionAndCountsItsBytes(t *testing.T) {
	ctx := context.Background()
	fm := newTestManager(t)
	folder, err := fm.CreateFolder(ctx, files.Folder{Name: "Papers", Audience: "staff"})
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	f, err := fm.CreateFile(ctx, files.File{FolderID: folder.ID, Name: "budget.pdf"})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	for _, body := range []string{"first", "second draft"} {
		if _, err := fm.AddVersion(ctx, files.Version{FileID: f.ID, ContentType: "application/pdf", Content: []byte(body), UploadedBy: "actor-1"}); err != nil {
			t.Fatalf("AddVersion: %v", err)
		}
	}
	got, err := fm.GetFile(ctx, f.ID)
	if err != nil || got.CurrentVersion != 2 {
		t.Fatalf("GetFile = %+v, %v; want current version 2", got, err)
	}
	vs, err := fm.ListVersions(ctx, f.ID)
	if err != nil || len(vs) != 2 {
		t.Fatalf("ListVersions = %d, %v", len(vs), err)
	}
	if vs[0].Number != 2 || vs[0].Size != int64(len("second draft")) || vs[0].UploadedBy != "actor-1" || len(vs[0].Checksum) != 64 {
		t.Fatalf("newest version = %+v", vs[0])
	}
}

func TestAnUploadWithNoCallerIsRefused(t *testing.T) {
	ctx := context.Background()
	fm := newTestManager(t)
	folder, _ := fm.CreateFolder(ctx, files.Folder{Name: "Papers", Audience: "public"})
	f, _ := fm.CreateFile(ctx, files.File{FolderID: folder.ID, Name: "a.txt"})
	if _, err := fm.AddVersion(ctx, files.Version{FileID: f.ID, ContentType: "text/plain", Content: []byte("x")}); err != files.ErrNoUploader {
		t.Fatalf("err = %v, want ErrNoUploader", err)
	}
}

func TestAFolderAudienceIsDeclaredData(t *testing.T) {
	ctx := context.Background()
	fm := newTestManager(t)
	if _, err := fm.CreateFolder(ctx, files.Folder{Name: "x", Audience: "everyone"}); err == nil {
		t.Fatal("an undeclared audience was accepted")
	}
	if _, err := fm.CreateFolder(ctx, files.Folder{Name: "x", Audience: "groups"}); err == nil {
		t.Fatal("a groups folder naming no group was accepted")
	}
	g, err := fm.CreateFolder(ctx, files.Folder{Name: "x", Audience: "groups", GroupIDs: []string{"g-1"}})
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	list, err := fm.ListFolders(ctx, files.FolderFilter{Audience: "groups"})
	if err != nil || len(list) != 1 || list[0].ID != g.ID || len(list[0].GroupIDs) != 1 {
		t.Fatalf("ListFolders = %+v, %v", list, err)
	}
}

func TestAFileNeedsAFolderThatExists(t *testing.T) {
	fm := newTestManager(t)
	if _, err := fm.CreateFile(context.Background(), files.File{FolderID: "nope", Name: "a"}); err != files.ErrFolderNotFound {
		t.Fatalf("err = %v, want ErrFolderNotFound", err)
	}
}

func TestEveryExampleFitsTheTypes(t *testing.T) {
	for _, p := range []string{"files.wakala.json", "spec/examples/council.files.json", "spec/examples/firm.files.json"} {
		s, err := files.LoadSpec(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err := files.Provision(db, s); err != nil {
			t.Fatalf("%s: Provision: %v", p, err)
		}
		disk, _ := files.NewDiskStore(t.TempDir())
		if _, err := files.NewFileManager(db, s, disk); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
}
