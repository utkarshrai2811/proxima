package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
)

var hookNames = []string{"onRequest", "onResponse", "onIntercept", "onFuzzResult"}

var (
	exportVarRe  = regexp.MustCompile(`(?m)^(\s*)export\s+(?:const|let|var)\s+`)
	exportFuncRe = regexp.MustCompile(`(?m)^(\s*)export\s+function\s+`)
	exportDefRe  = regexp.MustCompile(`(?m)^(\s*)export\s+default\s+`)
)

// transformExports rewrites ES-module export syntax so Goja (ES5.1 + partial
// ES6, no modules) can run the script and we can read the bindings as globals.
func transformExports(src string) string {
	src = exportVarRe.ReplaceAllString(src, "${1}var ")
	src = exportFuncRe.ReplaceAllString(src, "${1}function ")
	src = exportDefRe.ReplaceAllString(src, "${1}var __default = ")

	return src
}

func strOr(v interface{}, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}

	return def
}

// LoadPlugin reads a .js file, runs it in a fresh Goja VM, and extracts its
// meta and hook functions.
func LoadPlugin(filePath string, logger Logger, notifications *NotificationQueue) (*Plugin, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("plugin: read %s: %w", filePath, err)
	}

	p := &Plugin{
		Name:     filepath.Base(filePath),
		FilePath: filePath,
		LoadedAt: time.Now(),
		hooks:    map[string]goja.Callable{},
	}

	vm := goja.New()
	p.vm = vm
	setupProximaGlobal(vm, p, logger, notifications)

	if _, err := vm.RunString(transformExports(string(src))); err != nil {
		return nil, fmt.Errorf("plugin: run %s: %w", filePath, err)
	}

	if metaVal := vm.Get("meta"); metaVal != nil && !goja.IsUndefined(metaVal) && !goja.IsNull(metaVal) {
		if meta, ok := metaVal.Export().(map[string]interface{}); ok {
			p.Name = strOr(meta["name"], p.Name)
			p.Version = strOr(meta["version"], "")
			p.Description = strOr(meta["description"], "")
			p.Author = strOr(meta["author"], "")
		}
	}

	for _, name := range hookNames {
		if v := vm.Get(name); v != nil {
			if fn, ok := goja.AssertFunction(v); ok {
				p.hooks[name] = fn
			}
		}
	}

	return p, nil
}

// LoadAll loads every *.js file in dir. A file that fails to load is recorded
// as a disabled plugin carrying its LastError rather than aborting the scan.
func LoadAll(dir string, logger Logger, notifications *NotificationQueue) ([]*Plugin, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("plugin: read dir %s: %w", dir, err)
	}

	var plugins []*Plugin

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
			continue
		}

		path := filepath.Join(dir, e.Name())

		p, err := LoadPlugin(path, logger, notifications)
		if err != nil {
			if logger != nil {
				logger.Infow("plugin load failed", "file", path, "error", err.Error())
			}

			plugins = append(plugins, &Plugin{
				Name: e.Name(), FilePath: path, LastError: err.Error(),
				LoadedAt: time.Now(), hooks: map[string]goja.Callable{},
			})

			continue
		}

		plugins = append(plugins, p)
	}

	return plugins, nil
}
