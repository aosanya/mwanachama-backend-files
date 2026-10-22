package routes_test

import (
	"encoding/json"
	"net/http"
	"testing"

	files "github.com/aosanya/mwanachama-backend-files"
	"github.com/aosanya/mwanachama-backend-files/routes"
)

func TestTheSettingsAreReadOverTheRouteTable(t *testing.T) {
	srv := newServer(t, "actor-1")
	res, err := http.Get(srv.URL + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var got files.Settings
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || got.PhotoLongestSide != 1600 || got.PhotosPerQuestion != 5 || got.LargestFileBytes != 20*1024*1024 || len(got.Effective) != 3 {
		t.Fatalf("GET /settings = %d %+v", res.StatusCode, got)
	}
}

func TestATooLargeFileAnswers413(t *testing.T) {
	if routes.Sentinels["ErrFileTooLarge"] != files.ErrFileTooLarge {
		t.Fatal("ErrFileTooLarge is not mapped")
	}
	srv := newServer(t, "actor-1")
	_, folder := post(t, srv, "/folders", map[string]any{"name": "Papers", "audience": "staff"})
	_, file := post(t, srv, "/folders/"+folder["id"].(string)+"/files", map[string]any{"name": "big.bin"})
	big := make([]byte, 20*1024*1024+1)
	status, _ := post(t, srv, "/files/"+file["id"].(string)+"/versions", map[string]any{"content_type": "application/octet-stream", "content": big})
	if status != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", status)
	}
}
