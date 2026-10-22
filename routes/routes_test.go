package routes_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	files "github.com/aosanya/mwanachama-backend-files"
	"github.com/aosanya/mwanachama-backend-files/routes"
)

type callerKey struct{}

func callerFrom(ctx context.Context) string {
	id, _ := ctx.Value(callerKey{}).(string)
	return id
}

func newManager(t *testing.T) files.FileManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	s, err := files.SpecFor("routestest")
	if err != nil {
		t.Fatal(err)
	}
	if err := files.Provision(db, s); err != nil {
		t.Fatal(err)
	}
	disk, _ := files.NewDiskStore(t.TempDir())
	fm, err := files.NewFileManager(db, s, disk)
	if err != nil {
		t.Fatal(err)
	}
	return fm
}

func newServer(t *testing.T, caller string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(newManager(t), routes.Mount{Caller: callerFrom}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mux.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), callerKey{}, caller)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, srv *httptest.Server, path string, body any) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	res, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestNoRouteIsAnonymous(t *testing.T) {
	split := routes.Split(newManager(t), routes.Mount{})
	if len(split.Anonymous) != 0 {
		t.Fatalf("%d routes answer without a caller; every address is gated", len(split.Anonymous))
	}
	if len(routes.AnonymousActions) != 0 {
		t.Fatalf("AnonymousActions = %v, want none", routes.AnonymousActions)
	}
}

func TestTheUploaderComesFromTheCallerNotTheBody(t *testing.T) {
	srv := newServer(t, "actor-1")
	code, folder := post(t, srv, "/folders", map[string]any{"name": "Papers", "audience": "staff", "created_by": "someone-else"})
	if code != http.StatusCreated || folder["created_by"] != "actor-1" {
		t.Fatalf("create folder = %d %v", code, folder)
	}
	code, file := post(t, srv, "/folders/"+folder["id"].(string)+"/files", map[string]any{"name": "a.pdf", "folder_id": "elsewhere"})
	if code != http.StatusCreated || file["folder_id"] != folder["id"] {
		t.Fatalf("create file = %d %v", code, file)
	}
	code, v := post(t, srv, "/files/"+file["id"].(string)+"/versions", map[string]any{
		"content_type": "application/pdf",
		"content":      base64.StdEncoding.EncodeToString([]byte("pdf bytes")),
		"uploaded_by":  "someone-else",
	})
	if code != http.StatusCreated || v["uploaded_by"] != "actor-1" || v["number"] != float64(1) {
		t.Fatalf("add version = %d %v", code, v)
	}
	if _, leaked := v["storage_key"]; leaked {
		t.Fatal("the storage key reached the response")
	}
}

func TestAnUploadWithNoCallerIsUnauthorized(t *testing.T) {
	srv := newServer(t, "")
	_, folder := post(t, srv, "/folders", map[string]any{"name": "Papers", "audience": "public"})
	_, file := post(t, srv, "/folders/"+folder["id"].(string)+"/files", map[string]any{"name": "a.txt"})
	code, _ := post(t, srv, "/files/"+file["id"].(string)+"/versions", map[string]any{
		"content_type": "text/plain",
		"content":      base64.StdEncoding.EncodeToString([]byte("x")),
	})
	if code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", code)
	}
}
