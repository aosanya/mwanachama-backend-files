package files_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	files "github.com/aosanya/mwanachama-backend-files"
)

func TestTheDiskStoreHoldsWhatItWasGiven(t *testing.T) {
	ctx := context.Background()
	d, err := files.NewDiskStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Put(ctx, "k1", strings.NewReader("hello")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	r, err := d.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, _ := io.ReadAll(r)
	r.Close()
	if string(got) != "hello" {
		t.Fatalf("Get = %q", got)
	}
	if err := d.Delete(ctx, "k1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := d.Get(ctx, "k1"); !errors.Is(err, files.ErrBytesNotFound) {
		t.Fatalf("Get after Delete = %v", err)
	}
}

func TestTheDiskStoreRefusesAKeyThatLeavesItsDirectory(t *testing.T) {
	d, _ := files.NewDiskStore(t.TempDir())
	for _, key := range []string{"../escape", "a/b", "", ".hidden"} {
		if err := d.Put(context.Background(), key, strings.NewReader("x")); !errors.Is(err, files.ErrInvalidKey) {
			t.Errorf("Put(%q) = %v, want ErrInvalidKey", key, err)
		}
	}
}
