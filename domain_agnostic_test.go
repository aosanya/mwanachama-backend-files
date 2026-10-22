package files_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	files "github.com/aosanya/mwanachama-backend-files"
)

var domainWords = []string{
	"agency", "agencies", "wakala", "studio", "operator", "tenant", "resident",
	"council", "ward", "pothole", "matter", "tender", "client", "firm", "chambers",
	"survey", "questionnaire", "answer", "inspector",
}

func carriesDomainWord(name string) (string, bool) {
	lower := strings.ToLower(name)
	for _, w := range domainWords {
		if strings.Contains(lower, w) {
			return w, true
		}
	}
	return "", false
}

func TestNoDomainWordsInIdentifiers(t *testing.T) {
	for _, dir := range []string{"models", "routes", "."} {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		seen := map[string]bool{}
		for _, pkg := range pkgs {
			for path, file := range pkg.Files {
				ast.Inspect(file, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					w, bad := carriesDomainWord(id.Name)
					if !bad {
						return true
					}
					key := filepath.Base(path) + ":" + id.Name
					if seen[key] {
						return true
					}
					seen[key] = true
					t.Errorf("%s: identifier %q carries the domain word %q — what a file is in one domain belongs to that domain's spec",
						filepath.Base(path), id.Name, w)
					return true
				})
			}
		}
	}
}

func TestNoDomainWordsInTheBlueprint(t *testing.T) {
	b, err := files.Blueprint()
	if err != nil {
		t.Fatalf("blueprint: %v", err)
	}
	report := func(kind, name string) {
		if w, bad := carriesDomainWord(name); bad {
			t.Errorf("blueprint %s %q carries the domain word %q — it belongs in a domain's own spec", kind, name, w)
		}
	}
	for _, o := range b.Objects {
		report("role", o.Role)
		if o.Name != "" || o.Table != "" {
			t.Errorf("blueprint role %q names a table, which is the domain's to choose", o.Role)
		}
		for _, f := range o.Fields {
			report("field", o.Role+"."+f.Name)
			for _, v := range f.Values {
				report("stored value", o.Role+"."+f.Name+"="+v)
			}
		}
		for _, idx := range o.Indexes {
			report("index", o.Role+"."+idx.Name)
		}
	}
}
