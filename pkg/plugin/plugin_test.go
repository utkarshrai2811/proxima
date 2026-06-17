package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenCommandPerOS(t *testing.T) {
	t.Parallel()

	want := map[string]string{"darwin": "open", "windows": "explorer", "linux": "xdg-open"}[runtime.GOOS]
	if want == "" {
		want = "xdg-open"
	}

	cmd := openCommand("/tmp/x")
	if len(cmd.Args) == 0 || cmd.Args[0] != want {
		t.Fatalf("openCommand on %s = %v, want first arg %q", runtime.GOOS, cmd.Args, want)
	}
}

func TestTransformExports(t *testing.T) {
	t.Parallel()

	in := "export const meta = {};\nexport function onRequest(ctx) { return ctx; }\n"
	out := transformExports(in)

	if want := "var meta = {};\nfunction onRequest(ctx) { return ctx; }\n"; out != want {
		t.Fatalf("transformExports =\n%q\nwant\n%q", out, want)
	}
}

func TestIsLoopbackURL(t *testing.T) {
	t.Parallel()

	for _, u := range []string{"http://localhost/x", "http://127.0.0.1:8080/", "http://[::1]/", "not a url"} {
		if !isLoopbackURL(u) {
			t.Errorf("isLoopbackURL(%q) = false, want true", u)
		}
	}

	if isLoopbackURL("https://example.com/") {
		t.Error("isLoopbackURL(example.com) = true, want false")
	}
}

func writePlugin(t *testing.T, dir, name, src string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAndCallHook(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writePlugin(t, dir, "inject.js", `
		export const meta = { name: "Injector", version: "1.0.0", author: "Proxima User" };
		export function onRequest(ctx) {
		  ctx.request.headers['X-Test'] = 'yes';
		  proxima.log('ran');
		  return ctx;
		}
	`)

	mgr := NewManager(dir, nil, nil)
	if err := mgr.LoadAll(); err != nil {
		t.Fatal(err)
	}

	infos := mgr.List()
	if len(infos) != 1 || infos[0].Name != "Injector" || !infos[0].Enabled {
		t.Fatalf("List = %+v", infos)
	}

	ctx := mgr.CallHook("onRequest", &HookContext{
		Request: &RequestCtx{Method: "GET", URL: "http://x", Headers: map[string]string{}},
	})

	if ctx.Request == nil || ctx.Request.Headers["X-Test"] != "yes" {
		t.Fatalf("hook did not inject header: %+v", ctx.Request)
	}
}

func TestHookErrorRecorded(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writePlugin(t, dir, "boom.js", `
		export const meta = { name: "Boom" };
		export function onRequest(ctx) { throw new Error("boom"); }
	`)

	mgr := NewManager(dir, nil, nil)
	if err := mgr.LoadAll(); err != nil {
		t.Fatal(err)
	}

	// Must not panic/crash; the original context is returned unchanged.
	ctx := mgr.CallHook("onRequest", &HookContext{Request: &RequestCtx{Method: "GET"}})
	if ctx.Request == nil || ctx.Request.Method != "GET" {
		t.Fatalf("context corrupted by failing hook: %+v", ctx)
	}

	if infos := mgr.List(); len(infos) != 1 || infos[0].LastError == "" {
		t.Fatalf("expected LastError to be recorded: %+v", infos)
	}
}
