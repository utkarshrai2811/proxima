package main

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/utkarshrai2811/proxima/pkg/fuzzer"
	"github.com/utkarshrai2811/proxima/pkg/plugin"
	"github.com/utkarshrai2811/proxima/pkg/proxy"
)

func headerToMap(h http.Header) map[string]string {
	m := make(map[string]string, len(h))
	for k := range h {
		m[k] = h.Get(k)
	}

	return m
}

// requestToCtx builds a plugin RequestCtx, reading and restoring the body. The
// original body is returned so the caller can detect plugin modifications.
func requestToCtx(req *http.Request) (*plugin.RequestCtx, string) {
	body := ""

	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(data))
		body = string(data)
	}

	return &plugin.RequestCtx{
		Method: req.Method, URL: req.URL.String(), Headers: headerToMap(req.Header), Body: body,
	}, body
}

func applyRequestCtx(req *http.Request, rc *plugin.RequestCtx, original string) {
	if rc == nil {
		return
	}

	for k, v := range rc.Headers {
		req.Header.Set(k, v)
	}

	if rc.Body != original {
		data := []byte(rc.Body)
		req.Body = io.NopCloser(bytes.NewReader(data))
		req.ContentLength = int64(len(data))
		req.Header.Set("Content-Length", strconv.Itoa(len(data)))
	}
}

func applyResponseCtx(res *http.Response, rc *plugin.ResponseCtx, original string) {
	if rc == nil {
		return
	}

	if rc.StatusCode > 0 {
		res.StatusCode = rc.StatusCode
	}

	for k, v := range rc.Headers {
		res.Header.Set(k, v)
	}

	if rc.Body != original {
		data := []byte(rc.Body)
		res.Body = io.NopCloser(bytes.NewReader(data))
		res.ContentLength = int64(len(data))
		res.Header.Set("Content-Length", strconv.Itoa(len(data)))
	}
}

func pluginRequestModifier(mgr *plugin.Manager) proxy.RequestModifyMiddleware {
	return func(next proxy.RequestModifyFunc) proxy.RequestModifyFunc {
		return func(req *http.Request) {
			rc, original := requestToCtx(req)
			ctx := mgr.CallHook("onRequest", &plugin.HookContext{Request: rc})
			applyRequestCtx(req, ctx.Request, original)
			next(req)
		}
	}
}

func pluginResponseModifier(mgr *plugin.Manager) proxy.ResponseModifyMiddleware {
	return func(next proxy.ResponseModifyFunc) proxy.ResponseModifyFunc {
		return func(res *http.Response) error {
			original := ""

			if res.Body != nil {
				data, _ := io.ReadAll(res.Body)
				res.Body = io.NopCloser(bytes.NewReader(data))
				original = string(data)
			}

			hctx := &plugin.HookContext{
				Response: &plugin.ResponseCtx{
					StatusCode: res.StatusCode, Headers: headerToMap(res.Header), Body: original,
				},
			}

			if res.Request != nil {
				hctx.Request = &plugin.RequestCtx{
					Method: res.Request.Method, URL: res.Request.URL.String(),
					Headers: headerToMap(res.Request.Header),
				}
			}

			hctx = mgr.CallHook("onResponse", hctx)
			applyResponseCtx(res, hctx.Response, original)

			return next(res)
		}
	}
}

func pluginInterceptHook(mgr *plugin.Manager) func(*http.Request) string {
	return func(req *http.Request) string {
		rc, _ := requestToCtx(req)
		ctx := mgr.CallHook("onIntercept", &plugin.HookContext{Request: rc})

		return ctx.Action
	}
}

func pluginFuzzResultHook(mgr *plugin.Manager) func(fuzzer.FuzzResult) {
	return func(r fuzzer.FuzzResult) {
		mgr.CallHook("onFuzzResult", &plugin.HookContext{
			Result: &plugin.FuzzResultCtx{
				AttackID: r.AttackID, RequestIndex: r.RequestIndex, PayloadValues: r.PayloadValues,
				StatusCode: r.StatusCode, ResponseSize: r.ResponseSize,
				ResponseTimeMs: r.ResponseTimeMs, IsError: r.IsError,
			},
		})
	}
}
