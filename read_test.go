package files_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	files "github.com/aosanya/mwanachama-backend-files"
)

var linkKey = []byte("0123456789abcdef0123456789abcdef")

type readFixture struct {
	fm     files.FileManager
	reader *files.Reader
	file   files.File
	now    time.Time
}

func newReadFixture(t *testing.T) *readFixture {
	t.Helper()
	ctx := context.Background()
	fx := &readFixture{fm: newTestManager(t), now: time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)}
	folder, err := fx.fm.CreateFolder(ctx, files.Folder{Name: "Papers", Audience: "members"})
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	fx.file, err = fx.fm.CreateFile(ctx, files.File{FolderID: folder.ID, Name: "budget.pdf"})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	for _, body := range []string{"first", "second"} {
		if _, err := fx.fm.AddVersion(ctx, files.Version{FileID: fx.file.ID, ContentType: "application/pdf", Content: []byte(body), UploadedBy: "staff-1"}); err != nil {
			t.Fatalf("AddVersion: %v", err)
		}
	}
	check := func(_ context.Context, caller string, f files.Folder, _ files.File) (files.Standing, error) {
		switch {
		case caller == "staff-1":
			return files.StandingStaff, nil
		case caller == "member-1" && f.Audience == "members":
			return files.StandingReader, nil
		}
		return files.StandingNone, nil
	}
	fx.reader, err = files.NewReader(fx.fm, check, linkKey, 5*time.Minute)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	fx.reader.Now = func() time.Time { return fx.now }
	return fx
}

func readAll(t *testing.T, fx *readFixture, token string) string {
	t.Helper()
	_, body, err := fx.reader.Open(context.Background(), token)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer body.Close()
	b, _ := io.ReadAll(body)
	return string(b)
}

func TestACallerTheHostRefusesSeesNotFound(t *testing.T) {
	fx := newReadFixture(t)
	ctx := context.Background()
	if _, err := fx.reader.Link(ctx, "stranger", fx.file.ID, 0); err != files.ErrFileNotFound {
		t.Fatalf("refused caller err = %v, want ErrFileNotFound", err)
	}
	if _, err := fx.reader.Link(ctx, "stranger", "no-such-file", 0); err != files.ErrFileNotFound {
		t.Fatalf("missing file err = %v, want ErrFileNotFound", err)
	}
	if _, err := fx.reader.Link(ctx, "", fx.file.ID, 0); err != files.ErrFileNotFound {
		t.Fatalf("no caller err = %v, want ErrFileNotFound", err)
	}
	if _, err := fx.reader.Versions(ctx, "stranger", fx.file.ID); err != files.ErrFileNotFound {
		t.Fatalf("refused caller versions err = %v, want ErrFileNotFound", err)
	}
}

func TestAMemberGetsTheLatestVersionOnly(t *testing.T) {
	fx := newReadFixture(t)
	ctx := context.Background()
	vs, err := fx.reader.Versions(ctx, "member-1", fx.file.ID)
	if err != nil || len(vs) != 1 || vs[0].Number != 2 {
		t.Fatalf("member versions = %+v, %v; want only version 2", vs, err)
	}
	link, err := fx.reader.Link(ctx, "member-1", fx.file.ID, 0)
	if err != nil || link.Version.Number != 2 {
		t.Fatalf("member latest link = %+v, %v", link, err)
	}
	if got := readAll(t, fx, link.Token); got != "second" {
		t.Fatalf("member read %q, want second", got)
	}
	if _, err := fx.reader.Link(ctx, "member-1", fx.file.ID, 1); err != files.ErrFileNotFound {
		t.Fatalf("member naming an old version err = %v, want ErrFileNotFound", err)
	}
}

func TestStaffCanNameAnyVersion(t *testing.T) {
	fx := newReadFixture(t)
	ctx := context.Background()
	vs, err := fx.reader.Versions(ctx, "staff-1", fx.file.ID)
	if err != nil || len(vs) != 2 {
		t.Fatalf("staff versions = %d, %v; want 2", len(vs), err)
	}
	link, err := fx.reader.Link(ctx, "staff-1", fx.file.ID, 1)
	if err != nil {
		t.Fatalf("staff link: %v", err)
	}
	if got := readAll(t, fx, link.Token); got != "first" {
		t.Fatalf("staff read %q, want first", got)
	}
	if _, err := fx.reader.Link(ctx, "staff-1", fx.file.ID, 9); err != files.ErrFileNotFound {
		t.Fatalf("staff naming no version err = %v, want ErrFileNotFound", err)
	}
}

func TestAnExpiredLinkIsRefused(t *testing.T) {
	fx := newReadFixture(t)
	link, err := fx.reader.Link(context.Background(), "member-1", fx.file.ID, 0)
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	fx.now = fx.now.Add(5 * time.Minute)
	if _, _, err := fx.reader.Open(context.Background(), link.Token); err != files.ErrLinkExpired {
		t.Fatalf("expired link err = %v, want ErrLinkExpired", err)
	}
}

func TestATamperedLinkIsRefused(t *testing.T) {
	fx := newReadFixture(t)
	ctx := context.Background()
	link, err := fx.reader.Link(ctx, "member-1", fx.file.ID, 0)
	if err != nil {
		t.Fatalf("Link: %v", err)
	}
	head, sig, _ := strings.Cut(link.Token, ".")
	forged := strings.Replace(link.Token, head, head+"x", 1)
	for name, token := range map[string]string{
		"altered claim":   forged,
		"no signature":    head,
		"wrong signature": head + "." + strings.Repeat("A", len(sig)),
		"empty":           "",
	} {
		if _, _, err := fx.reader.Open(ctx, token); err != files.ErrLinkInvalid {
			t.Fatalf("%s: err = %v, want ErrLinkInvalid", name, err)
		}
	}
	other, _ := files.NewReader(fx.fm, func(context.Context, string, files.Folder, files.File) (files.Standing, error) {
		return files.StandingStaff, nil
	}, []byte("ffffffffffffffffffffffffffffffff"), time.Minute)
	if _, _, err := other.Open(ctx, link.Token); err != files.ErrLinkInvalid {
		t.Fatalf("another key err = %v, want ErrLinkInvalid", err)
	}
}

func TestAReaderRefusesAShortKey(t *testing.T) {
	fm := newTestManager(t)
	check := func(context.Context, string, files.Folder, files.File) (files.Standing, error) {
		return files.StandingNone, nil
	}
	if _, err := files.NewReader(fm, check, []byte("short"), time.Minute); err == nil {
		t.Fatal("a short link key was accepted")
	}
}
