package plugin

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
)

// setupProximaGlobal installs the `proxima` global object into a plugin VM.
func setupProximaGlobal(vm *goja.Runtime, p *Plugin, logger Logger, notifications *NotificationQueue) {
	obj := vm.NewObject()

	_ = obj.Set("log", func(call goja.FunctionCall) goja.Value {
		if logger != nil {
			logger.Infow("plugin", "name", p.Name, "message", call.Argument(0).String())
		}

		return goja.Undefined()
	})

	_ = obj.Set("alert", func(call goja.FunctionCall) goja.Value {
		if notifications != nil {
			notifications.Push(p.Name, call.Argument(0).String())
		}

		return goja.Undefined()
	})

	httpObj := vm.NewObject()
	_ = httpObj.Set("send", buildHTTPSendFunc(vm))
	_ = obj.Set("http", httpObj)

	store := &sync.Map{}
	storeObj := vm.NewObject()
	_ = storeObj.Set("get", func(call goja.FunctionCall) goja.Value {
		if v, ok := store.Load(call.Argument(0).String()); ok {
			return vm.ToValue(v)
		}

		return goja.Null()
	})
	_ = storeObj.Set("set", func(call goja.FunctionCall) goja.Value {
		store.Store(call.Argument(0).String(), call.Argument(1).Export())

		return goja.Undefined()
	})
	_ = obj.Set("store", storeObj)

	_ = vm.Set("proxima", obj)
}

// buildHTTPSendFunc returns proxima.http.send. It blocks requests to loopback
// addresses to prevent SSRF against the local admin API.
func buildHTTPSendFunc(vm *goja.Runtime) func(goja.FunctionCall) goja.Value {
	client := &http.Client{Timeout: 10 * time.Second}

	return func(call goja.FunctionCall) goja.Value {
		spec, ok := call.Argument(0).Export().(map[string]interface{})
		if !ok {
			panic(vm.ToValue("proxima.http.send: expected a request object"))
		}

		method, _ := spec["method"].(string)
		if method == "" {
			method = http.MethodGet
		}

		rawURL, _ := spec["url"].(string)
		if isLoopbackURL(rawURL) {
			panic(vm.ToValue("proxima.http.send: requests to localhost are blocked"))
		}

		var body io.Reader
		if b, ok := spec["body"].(string); ok && b != "" {
			body = strings.NewReader(b)
		}

		req, err := http.NewRequest(method, rawURL, body)
		if err != nil {
			panic(vm.ToValue("proxima.http.send: " + err.Error()))
		}

		if hdrs, ok := spec["headers"].(map[string]interface{}); ok {
			for k, v := range hdrs {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			panic(vm.ToValue("proxima.http.send: " + err.Error()))
		}
		defer resp.Body.Close()

		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

		headers := map[string]interface{}{}
		for k := range resp.Header {
			headers[k] = resp.Header.Get(k)
		}

		return vm.ToValue(map[string]interface{}{
			"statusCode": resp.StatusCode,
			"headers":    headers,
			"body":       string(data),
		})
	}
}

// isLoopbackURL reports whether the URL targets a loopback host (anti-SSRF).
func isLoopbackURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return true // block anything we cannot parse
	}

	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return false
}
