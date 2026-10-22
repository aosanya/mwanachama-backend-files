package files_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	files "github.com/aosanya/mwanachama-backend-files"
)

type heldBytes struct {
	mu   sync.Mutex
	held map[string][]byte
	last []byte
}

func (h *heldBytes) Put(_ context.Context, key string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.held[key] = b
	h.last = b
	return nil
}

func (h *heldBytes) Get(_ context.Context, key string) (io.ReadCloser, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	b, ok := h.held[key]
	if !ok {
		return nil, files.ErrBytesNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (h *heldBytes) Delete(_ context.Context, key string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.held, key)
	return nil
}

func newPhotoManager(t *testing.T) (files.FileManager, *heldBytes, string) {
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
	held := &heldBytes{held: map[string][]byte{}}
	fm, err := files.NewFileManager(db, s, held)
	if err != nil {
		t.Fatalf("NewFileManager: %v", err)
	}
	ctx := context.Background()
	folder, err := fm.CreateFolder(ctx, files.Folder{Name: "Answers", Audience: "staff"})
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	f, err := fm.CreateFile(ctx, files.File{FolderID: folder.ID, Name: "photo"})
	if err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	return fm, held, f.ID
}

type ifdEntry struct {
	tag, kind uint16
	count     uint32
	value     uint32
}

func ifd(entries []ifdEntry) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint16(len(entries)))
	for _, e := range entries {
		binary.Write(&b, binary.LittleEndian, e.tag)
		binary.Write(&b, binary.LittleEndian, e.kind)
		binary.Write(&b, binary.LittleEndian, e.count)
		binary.Write(&b, binary.LittleEndian, e.value)
	}
	binary.Write(&b, binary.LittleEndian, uint32(0))
	return b.Bytes()
}

func exifPayload(withTime, withPlace bool) []byte {
	var ifd0 []ifdEntry
	ifd0Len := func(n int) int { return 2 + 12*n + 4 }
	n := 0
	if withTime {
		n++
	}
	if withPlace {
		n++
	}
	next := uint32(8 + ifd0Len(n))
	var tail bytes.Buffer
	if withTime {
		exifAt := next + uint32(tail.Len())
		ifd0 = append(ifd0, ifdEntry{0x8769, 4, 1, exifAt})
		dateAt := exifAt + uint32(ifd0Len(1))
		tail.Write(ifd([]ifdEntry{{0x9003, 2, 20, dateAt}}))
		tail.WriteString("2026:10:22 09:30:00\x00")
	}
	if withPlace {
		gpsAt := next + uint32(tail.Len())
		ifd0 = append(ifd0, ifdEntry{0x8825, 4, 1, gpsAt})
		ratAt := gpsAt + uint32(ifd0Len(4))
		tail.Write(ifd([]ifdEntry{
			{0x0001, 2, 2, uint32('S')},
			{0x0002, 5, 3, ratAt},
			{0x0003, 2, 2, uint32('E')},
			{0x0004, 5, 3, ratAt},
		}))
		for i := 0; i < 6; i++ {
			binary.Write(&tail, binary.LittleEndian, uint32(1))
		}
	}
	var t bytes.Buffer
	t.WriteString("II")
	binary.Write(&t, binary.LittleEndian, uint16(42))
	binary.Write(&t, binary.LittleEndian, uint32(8))
	t.Write(ifd(ifd0))
	t.Write(tail.Bytes())
	return append([]byte("Exif\x00\x00"), t.Bytes()...)
}

func segment(marker byte, payload []byte) []byte {
	out := []byte{0xFF, marker, 0, 0}
	binary.BigEndian.PutUint16(out[2:], uint16(len(payload)+2))
	return append(out, payload...)
}

func jpeg(exif []byte) []byte {
	var b bytes.Buffer
	b.Write([]byte{0xFF, 0xD8})
	b.Write(segment(0xE0, []byte("JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")))
	if exif != nil {
		b.Write(segment(0xE1, exif))
	}
	b.Write(segment(0xDB, bytes.Repeat([]byte{1}, 65)))
	b.Write(segment(0xDA, []byte{1, 1, 0, 0, 0x3F, 0}))
	b.Write([]byte{0x12, 0x34, 0x56})
	b.Write([]byte{0xFF, 0xD9})
	return b.Bytes()
}

func TestTheStoredContentTypeIsReadFromTheBytes(t *testing.T) {
	fm, _, fileID := newPhotoManager(t)
	v, err := fm.AddVersion(context.Background(), files.Version{FileID: fileID, Name: "report.pdf", ContentType: "application/pdf", Content: jpeg(nil), UploadedBy: "actor-1"})
	if err != nil {
		t.Fatalf("AddVersion: %v", err)
	}
	if v.ContentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", v.ContentType)
	}
}

func TestAPhotoLosesItsLocationDataUnlessTheQuestionNeedsIt(t *testing.T) {
	fm, held, fileID := newPhotoManager(t)
	photo := jpeg(exifPayload(true, true))
	if _, err := fm.AddVersion(context.Background(), files.Version{FileID: fileID, ContentType: "image/jpeg", Content: photo, UploadedBy: "actor-1"}); err != nil {
		t.Fatalf("AddVersion: %v", err)
	}
	if bytes.Contains(held.last, []byte("Exif\x00\x00")) {
		t.Fatalf("stored bytes still carry exif")
	}
	if !bytes.Equal(held.last, jpeg(nil)) {
		t.Fatalf("stored bytes are not the photo without its exif segment")
	}
}

func TestAPhotoTakenWithTimeAndPlaceIsKeptAsIs(t *testing.T) {
	fm, held, fileID := newPhotoManager(t)
	photo := jpeg(exifPayload(true, true))
	if _, err := fm.AddVersion(context.Background(), files.Version{FileID: fileID, ContentType: "image/jpeg", Content: photo, UploadedBy: "actor-1", RequireTimeAndPlace: true}); err != nil {
		t.Fatalf("AddVersion: %v", err)
	}
	if !bytes.Equal(held.last, photo) {
		t.Fatalf("a photo that needed its time and place was changed")
	}
}

func TestAPhotoMissingTimeOrPlaceIsRefusedWhenTheQuestionNeedsThem(t *testing.T) {
	cases := map[string][]byte{
		"no exif":  jpeg(nil),
		"no place": jpeg(exifPayload(true, false)),
		"no time":  jpeg(exifPayload(false, true)),
		"not jpeg": []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x00IEND\xaeB`\x82"),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			fm, held, fileID := newPhotoManager(t)
			_, err := fm.AddVersion(context.Background(), files.Version{FileID: fileID, ContentType: "image/jpeg", Content: content, UploadedBy: "actor-1", RequireTimeAndPlace: true})
			if !errors.Is(err, files.ErrTimeAndPlaceRequired) {
				t.Fatalf("err = %v, want ErrTimeAndPlaceRequired", err)
			}
			if len(held.held) != 0 {
				t.Fatalf("refused photo was stored")
			}
		})
	}
}

func pngChunk(kind string, data []byte) []byte {
	out := make([]byte, 8, 12+len(data))
	binary.BigEndian.PutUint32(out, uint32(len(data)))
	copy(out[4:], kind)
	out = append(out, data...)
	return append(out, 0, 0, 0, 0)
}

func TestAPNGLosesItsExifChunk(t *testing.T) {
	fm, held, fileID := newPhotoManager(t)
	head := append([]byte("\x89PNG\r\n\x1a\n"), pngChunk("IHDR", make([]byte, 13))...)
	tail := pngChunk("IEND", nil)
	photo := append(append(append([]byte{}, head...), pngChunk("eXIf", exifPayload(true, true)[6:])...), tail...)
	v, err := fm.AddVersion(context.Background(), files.Version{FileID: fileID, ContentType: "image/png", Content: photo, UploadedBy: "actor-1"})
	if err != nil {
		t.Fatalf("AddVersion: %v", err)
	}
	if v.ContentType != "image/png" || !bytes.Equal(held.last, append(head, tail...)) {
		t.Fatalf("stored png still carries its exif chunk")
	}
}

func riffChunk(kind string, data []byte) []byte {
	out := make([]byte, 8, 9+len(data))
	copy(out, kind)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(data)))
	out = append(out, data...)
	if len(data)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

func webp(chunks ...[]byte) []byte {
	var body []byte
	for _, c := range chunks {
		body = append(body, c...)
	}
	out := []byte("RIFF\x00\x00\x00\x00WEBP")
	binary.LittleEndian.PutUint32(out[4:], uint32(4+len(body)))
	return append(out, body...)
}

func TestAWebPLosesItsExifChunkAndFlag(t *testing.T) {
	fm, held, fileID := newPhotoManager(t)
	flagged := make([]byte, 10)
	flagged[0] = 0x08
	cleared := make([]byte, 10)
	image := riffChunk("VP8L", []byte{0x2f, 0, 0, 0, 0})
	photo := webp(riffChunk("VP8X", flagged), image, riffChunk("EXIF", exifPayload(true, true)[6:]))
	v, err := fm.AddVersion(context.Background(), files.Version{FileID: fileID, ContentType: "image/webp", Content: photo, UploadedBy: "actor-1"})
	if err != nil {
		t.Fatalf("AddVersion: %v", err)
	}
	want := webp(riffChunk("VP8X", cleared), image)
	if v.ContentType != "image/webp" || !bytes.Equal(held.last, want) {
		t.Fatalf("stored webp still carries its exif chunk or flag (type %q)", v.ContentType)
	}
}
