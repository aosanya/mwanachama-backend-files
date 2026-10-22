package files

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Standing int

const (
	StandingNone Standing = iota
	StandingReader
	StandingStaff
)

type ReadCheck func(ctx context.Context, caller string, folder Folder, file File) (Standing, error)

var (
	ErrLinkInvalid = errors.New("download link is not valid")
	ErrLinkExpired = errors.New("download link has expired")
)

const minLinkKey = 32

type Reader struct {
	files    *fileManager
	check    ReadCheck
	key      []byte
	lifetime time.Duration
	Now      func() time.Time
}

type DownloadLink struct {
	Token     string  `json:"token"`
	ExpiresAt string  `json:"expires_at"`
	Version   Version `json:"version"`
}

func NewReader(fm FileManager, check ReadCheck, key []byte, lifetime time.Duration) (*Reader, error) {
	m, ok := fm.(*fileManager)
	if !ok {
		return nil, fmt.Errorf("NewReader: the manager must come from NewFileManager")
	}
	if check == nil {
		return nil, fmt.Errorf("NewReader: a read check is required")
	}
	if len(key) < minLinkKey {
		return nil, fmt.Errorf("NewReader: the link key must be at least %d bytes", minLinkKey)
	}
	if lifetime <= 0 {
		return nil, fmt.Errorf("NewReader: a link lifetime is required")
	}
	return &Reader{files: m, check: check, key: append([]byte(nil), key...), lifetime: lifetime, Now: time.Now}, nil
}

func (r *Reader) Versions(ctx context.Context, caller, fileID string) ([]Version, error) {
	standing, err := r.standing(ctx, caller, fileID)
	if err != nil {
		return nil, err
	}
	vs, err := r.files.ListVersions(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if standing == StandingStaff || len(vs) == 0 {
		return vs, nil
	}
	return vs[:1], nil
}

func (r *Reader) Link(ctx context.Context, caller, fileID string, number int) (DownloadLink, error) {
	vs, err := r.Versions(ctx, caller, fileID)
	if err != nil {
		return DownloadLink{}, err
	}
	v, ok := pickVersion(vs, number)
	if !ok {
		return DownloadLink{}, ErrFileNotFound
	}
	expires := r.Now().UTC().Add(r.lifetime).Truncate(time.Second)
	return DownloadLink{
		Token:     r.sign(v.FileID, v.Number, expires.Unix()),
		ExpiresAt: expires.Format(time.RFC3339),
		Version:   v,
	}, nil
}

func (r *Reader) Open(ctx context.Context, token string) (Version, io.ReadCloser, error) {
	fileID, number, expires, err := r.verify(token)
	if err != nil {
		return Version{}, nil, err
	}
	if !r.Now().UTC().Before(time.Unix(expires, 0)) {
		return Version{}, nil, ErrLinkExpired
	}
	vs, err := r.files.ListVersions(ctx, fileID)
	if err != nil {
		return Version{}, nil, ErrFileNotFound
	}
	v, ok := pickVersion(vs, number)
	if !ok {
		return Version{}, nil, ErrFileNotFound
	}
	body, err := r.files.bytes.Get(ctx, v.StorageKey)
	if err != nil {
		return Version{}, nil, fmt.Errorf("Open: %w", err)
	}
	return v, body, nil
}

func (r *Reader) standing(ctx context.Context, caller, fileID string) (Standing, error) {
	if caller == "" {
		return StandingNone, ErrFileNotFound
	}
	f, err := r.files.GetFile(ctx, fileID)
	if err != nil {
		return StandingNone, err
	}
	folder, err := r.files.getFolder(ctx, f.FolderID)
	if err != nil {
		return StandingNone, ErrFileNotFound
	}
	standing, err := r.check(ctx, caller, folder, f)
	if err != nil {
		return StandingNone, err
	}
	if standing != StandingReader && standing != StandingStaff {
		return StandingNone, ErrFileNotFound
	}
	return standing, nil
}

func pickVersion(vs []Version, number int) (Version, bool) {
	if len(vs) == 0 {
		return Version{}, false
	}
	if number == 0 {
		return vs[0], true
	}
	for _, v := range vs {
		if v.Number == number {
			return v, true
		}
	}
	return Version{}, false
}

func (r *Reader) sign(fileID string, number int, expires int64) string {
	claim := fileID + "|" + strconv.Itoa(number) + "|" + strconv.FormatInt(expires, 10)
	enc := base64.RawURLEncoding
	return enc.EncodeToString([]byte(claim)) + "." + enc.EncodeToString(r.mac(claim))
}

func (r *Reader) mac(claim string) []byte {
	h := hmac.New(sha256.New, r.key)
	h.Write([]byte(claim))
	return h.Sum(nil)
}

func (r *Reader) verify(token string) (string, int, int64, error) {
	enc := base64.RawURLEncoding
	head, tail, ok := strings.Cut(token, ".")
	if !ok {
		return "", 0, 0, ErrLinkInvalid
	}
	claim, err := enc.DecodeString(head)
	if err != nil {
		return "", 0, 0, ErrLinkInvalid
	}
	sig, err := enc.DecodeString(tail)
	if err != nil || !hmac.Equal(sig, r.mac(string(claim))) {
		return "", 0, 0, ErrLinkInvalid
	}
	parts := strings.Split(string(claim), "|")
	if len(parts) != 3 {
		return "", 0, 0, ErrLinkInvalid
	}
	number, err := strconv.Atoi(parts[1])
	if err != nil || number < 1 {
		return "", 0, 0, ErrLinkInvalid
	}
	expires, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", 0, 0, ErrLinkInvalid
	}
	return parts[0], number, expires, nil
}
