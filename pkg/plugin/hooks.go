package plugin

import (
	"fmt"
	"time"

	"github.com/dop251/goja"
)

const hookTimeout = 5 * time.Second

// CallHook invokes the named hook on every enabled plugin in order. A plugin
// that errors, panics, or times out has its LastError recorded and is skipped;
// the proxy is never crashed. The (possibly modified) context is threaded
// through and returned.
func (m *Manager) CallHook(hookName string, ctx *HookContext) *HookContext {
	for _, p := range m.enabledPlugins() {
		p.mu.Lock()

		hook, ok := p.hooks[hookName]
		if !ok {
			p.mu.Unlock()

			continue
		}

		result, err := callWithTimeout(p, hook, ctx)
		if err != nil {
			p.LastError = err.Error()
		}

		p.mu.Unlock()

		if err != nil {
			if m.logger != nil {
				m.logger.Infow("plugin hook failed", "plugin", p.Name, "hook", hookName, "error", err.Error())
			}

			continue
		}

		if result != nil {
			ctx = result
		}
	}

	return ctx
}

// callWithTimeout runs one hook with a hard timeout, recovering from panics.
// The caller holds p.mu.
func callWithTimeout(p *Plugin, hook goja.Callable, ctx *HookContext) (out *HookContext, err error) {
	defer func() {
		if r := recover(); r != nil {
			out = nil
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	timer := time.AfterFunc(hookTimeout, func() { p.vm.Interrupt("hook timeout") })
	defer timer.Stop()
	defer p.vm.ClearInterrupt()

	jsCtx := p.vm.ToValue(hookContextToMap(ctx))

	ret, callErr := hook(goja.Undefined(), jsCtx)
	if callErr != nil {
		return nil, callErr
	}

	val := ret
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		val = jsCtx
	}

	return mapToHookContext(val.Export()), nil
}

func toIfaceMap(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}

	return out
}

func hookContextToMap(ctx *HookContext) map[string]interface{} {
	m := map[string]interface{}{"action": ctx.Action}

	if ctx.Request != nil {
		m["request"] = map[string]interface{}{
			"method": ctx.Request.Method, "url": ctx.Request.URL,
			"headers": toIfaceMap(ctx.Request.Headers), "body": ctx.Request.Body,
		}
	}

	if ctx.Response != nil {
		m["response"] = map[string]interface{}{
			"statusCode": ctx.Response.StatusCode,
			"headers":    toIfaceMap(ctx.Response.Headers), "body": ctx.Response.Body,
		}
	}

	if ctx.Result != nil {
		m["result"] = map[string]interface{}{
			"attackId": ctx.Result.AttackID, "requestIndex": ctx.Result.RequestIndex,
			"payloadValues": toIfaceMap(ctx.Result.PayloadValues), "statusCode": ctx.Result.StatusCode,
			"responseSize": ctx.Result.ResponseSize, "responseTimeMs": ctx.Result.ResponseTimeMs,
			"isError": ctx.Result.IsError,
		}
	}

	return m
}

func asString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}

	return ""
}

func asInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

func asStringMap(v interface{}) map[string]string {
	out := map[string]string{}

	if m, ok := v.(map[string]interface{}); ok {
		for k, val := range m {
			out[k] = fmt.Sprintf("%v", val)
		}
	}

	return out
}

func mapToHookContext(v interface{}) *HookContext {
	m, ok := v.(map[string]interface{})
	if !ok {
		return &HookContext{}
	}

	ctx := &HookContext{Action: asString(m["action"])}

	if rm, ok := m["request"].(map[string]interface{}); ok {
		ctx.Request = &RequestCtx{
			Method: asString(rm["method"]), URL: asString(rm["url"]),
			Headers: asStringMap(rm["headers"]), Body: asString(rm["body"]),
		}
	}

	if rm, ok := m["response"].(map[string]interface{}); ok {
		ctx.Response = &ResponseCtx{
			StatusCode: asInt(rm["statusCode"]),
			Headers:    asStringMap(rm["headers"]), Body: asString(rm["body"]),
		}
	}

	return ctx
}
