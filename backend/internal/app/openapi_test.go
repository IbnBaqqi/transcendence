package app

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	// Aliased for the same reason as in docs.go: this package has a type
	// called api.
	openapi "github.com/IbnBaqqi/transcendence/api"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/storage"
)

// The methods we document.
var httpMethods = []string{"get", "post", "put", "patch", "delete"}

// docsPlumbing are routes that serve the documentation itself.
var docsPlumbing = []string{specPath, docsPath + "/*"}

// Only the parts of the spec this test needs.
type specDoc struct {
	Servers []specServer                    `yaml:"servers"`
	Paths   map[string]map[string]yaml.Node `yaml:"paths"`
}

type specServer struct {
	URL string `yaml:"url"`
}

// The spec and the router are two independent descriptions of the same API.
func TestSpecMatchesRouter(t *testing.T) {
	inSpec := operationsFromSpec(t)
	inRouter := operationsFromRouter(t)

	for _, op := range diff(inRouter, inSpec) {
		t.Errorf("routed but not documented: %s (add it to api/openapi.yaml)", op)
	}
	for _, op := range diff(inSpec, inRouter) {
		t.Errorf("documented but not routed: %s (stale entry in api/openapi.yaml)", op)
	}
}

func operationsFromSpec(t *testing.T) []string {
	t.Helper()

	var doc specDoc
	if err := yaml.Unmarshal(openapi.Spec, &doc); err != nil {
		t.Fatalf("parsing the spec: %v", err)
	}
	if len(doc.Servers) == 0 {
		t.Fatal("the spec declares no servers")
	}

	base := serverPath(t, doc.Servers[0].URL)

	var ops []string
	for path, item := range doc.Paths {
		prefix := base

		if node, ok := item["servers"]; ok {
			var servers []specServer
			if err := node.Decode(&servers); err != nil {
				t.Fatalf("parsing servers for %s: %v", path, err)
			}
			if len(servers) == 0 {
				t.Fatalf("%s declares an empty servers list", path)
			}
			prefix = serverPath(t, servers[0].URL)
		}

		for key := range item {
			if slices.Contains(httpMethods, key) {
				ops = append(ops, strings.ToUpper(key)+" "+prefix+path)
			}
		}
	}
	return ops
}

// serverPath pulls just the path out of a server URL:
// "http://localhost:8080/api/v1" -> "/api/v1".
func serverPath(t *testing.T, raw string) string {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing server url %q: %v", raw, err)
	}
	return strings.TrimSuffix(parsed.Path, "/")
}

// operationsFromRouter asks chi for its own route table.
func operationsFromRouter(t *testing.T) []string {
	t.Helper()

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	handler := NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)), // logs go nowhere
		&api{Files: files, DB: &database.DB{}},
	)

	routes, ok := handler.(chi.Routes)
	if !ok {
		t.Fatal("NewRouter no longer returns a chi router")
	}

	seen := map[string]bool{}
	err = chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if slices.Contains(docsPlumbing, route) {
			return nil
		}
		if !slices.Contains(httpMethods, strings.ToLower(method)) {
			return nil
		}
		seen[method+" "+normalize(route)] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walking the router: %v", err)
	}

	ops := make([]string, 0, len(seen))
	for op := range seen {
		ops = append(ops, op)
	}
	return ops
}

const uploadRoute = "/uploads/*"

// normalize rewrites chi's wildcard into the spec's parameter spelling.
func normalize(route string) string {
	if route == uploadRoute {
		return "/uploads/{filename}"
	}
	return route
}

// diff returns the entries of a that are missing from b.
func diff(a, b []string) []string {
	only := []string{}
	for _, x := range a {
		if !slices.Contains(b, x) {
			only = append(only, x)
		}
	}
	sort.Strings(only)
	return only
}
