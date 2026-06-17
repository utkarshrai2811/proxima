package reqlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/oklog/ulid"

	"github.com/utkarshrai2811/proxima/pkg/filter"
	"github.com/utkarshrai2811/proxima/pkg/log"
	"github.com/utkarshrai2811/proxima/pkg/proxy"
	"github.com/utkarshrai2811/proxima/pkg/scope"
)

type contextKey int

const (
	LogBypassedKey contextKey = iota
	ReqLogIDKey
)

var (
	ErrRequestNotFound    = errors.New("reqlog: request not found")
	ErrProjectIDMustBeSet = errors.New("reqlog: project ID must be set")
)

type RequestLog struct {
	ID        ulid.ULID
	ProjectID ulid.ULID

	URL    *url.URL
	Method string
	Proto  string
	Header http.Header
	Body   []byte
	// Truncated is true when Body was capped at the configured max body size.
	Truncated bool

	Response *ResponseLog
}

type ResponseLog struct {
	Proto      string
	StatusCode int
	Status     string
	Header     http.Header
	Body       []byte
	// Truncated is true when Body was capped at the configured max body size.
	Truncated bool
}

type Service struct {
	bypassOutOfScopeRequests bool
	findReqsFilter           FindRequestsFilter
	activeProjectID          ulid.ULID
	scope                    *scope.Scope
	repo                     Repository
	logger                   log.Logger
	maxBodySize              int64
}

type FindRequestsFilter struct {
	ProjectID   ulid.ULID
	OnlyInScope bool
	SearchExpr  filter.Expression
}

type Config struct {
	ActiveProjectID ulid.ULID
	Scope           *scope.Scope
	Repository      Repository
	Logger          log.Logger
	// MaxBodySize caps the number of request/response body bytes stored in the
	// log. A value <= 0 means no limit. The full body is still forwarded.
	MaxBodySize int64
}

func NewService(cfg Config) *Service {
	s := &Service{
		activeProjectID: cfg.ActiveProjectID,
		repo:            cfg.Repository,
		scope:           cfg.Scope,
		logger:          cfg.Logger,
		maxBodySize:     cfg.MaxBodySize,
	}

	if s.logger == nil {
		s.logger = log.NewNopLogger()
	}

	return s
}

func (svc *Service) FindRequests(ctx context.Context) ([]RequestLog, error) {
	return svc.repo.FindRequestLogs(ctx, svc.findReqsFilter, svc.scope)
}

func (svc *Service) FindRequestLogByID(ctx context.Context, id ulid.ULID) (RequestLog, error) {
	return svc.repo.FindRequestLogByID(ctx, svc.activeProjectID, id)
}

func (svc *Service) ClearRequests(ctx context.Context, projectID ulid.ULID) error {
	return svc.repo.ClearRequestLogs(ctx, projectID)
}

func (svc *Service) RequestModifier(next proxy.RequestModifyFunc) proxy.RequestModifyFunc {
	return func(req *http.Request) {
		next(req)

		clone := req.Clone(req.Context())

		var (
			body      []byte
			truncated bool
		)

		if req.Body != nil {
			logged, trunc, full, err := readBodyForLogging(req.Body, svc.maxBodySize)
			if err != nil {
				svc.logger.Errorw("Failed to read request body for logging.",
					"error", err)
				return
			}

			body = logged
			truncated = trunc
			// Restore the full body so it is forwarded to the target intact.
			req.Body = io.NopCloser(full)
			clone.Body = io.NopCloser(bytes.NewReader(logged))
		}

		// Bypass logging if no project is active.
		if svc.activeProjectID.Compare(ulid.ULID{}) == 0 {
			ctx := context.WithValue(req.Context(), LogBypassedKey, true)
			*req = *req.WithContext(ctx)

			svc.logger.Debugw("Bypassed logging: no active project.",
				"url", req.URL.String())

			return
		}

		// Bypass logging if this setting is enabled and the incoming request
		// doesn't match any scope rules.
		if svc.bypassOutOfScopeRequests && !svc.scope.Match(clone, body) {
			ctx := context.WithValue(req.Context(), LogBypassedKey, true)
			*req = *req.WithContext(ctx)

			svc.logger.Debugw("Bypassed logging: request doesn't match any scope rules.",
				"url", req.URL.String())

			return
		}

		reqID, ok := proxy.RequestIDFromContext(req.Context())
		if !ok {
			svc.logger.Errorw("Bypassed logging: request doesn't have an ID.")
			return
		}

		reqLog := RequestLog{
			ID:        reqID,
			ProjectID: svc.activeProjectID,
			Method:    clone.Method,
			URL:       clone.URL,
			Proto:     clone.Proto,
			Header:    clone.Header,
			Body:      body,
			Truncated: truncated,
		}

		err := svc.repo.StoreRequestLog(req.Context(), reqLog)
		if err != nil {
			svc.logger.Errorw("Failed to store request log.",
				"error", err)
			return
		}

		svc.logger.Debugw("Stored request log.",
			"reqLogID", reqLog.ID.String(),
			"url", reqLog.URL.String())

		ctx := context.WithValue(req.Context(), ReqLogIDKey, reqLog.ID)
		*req = *req.WithContext(ctx)
	}
}

func (svc *Service) ResponseModifier(next proxy.ResponseModifyFunc) proxy.ResponseModifyFunc {
	return func(res *http.Response) error {
		if err := next(res); err != nil {
			return err
		}

		if bypassed, _ := res.Request.Context().Value(LogBypassedKey).(bool); bypassed {
			return nil
		}

		reqLogID, ok := res.Request.Context().Value(ReqLogIDKey).(ulid.ULID)
		if !ok {
			return errors.New("reqlog: request is missing ID")
		}

		resLog := ResponseLog{
			Proto:      res.Proto,
			StatusCode: res.StatusCode,
			Status:     res.Status,
			Header:     res.Header,
		}

		if res.Body != nil {
			logged, truncated, full, err := readBodyForLogging(res.Body, svc.maxBodySize)
			if err != nil {
				return fmt.Errorf("reqlog: could not read response body: %w", err)
			}

			resLog.Body = logged
			resLog.Truncated = truncated
			// Restore the full body so it is forwarded to the client intact.
			res.Body = io.NopCloser(full)
		}

		go func() {
			if err := svc.repo.StoreResponseLog(context.Background(), svc.activeProjectID, reqLogID, resLog); err != nil {
				svc.logger.Errorw("Failed to store response log.",
					"error", err)
			} else {
				svc.logger.Debugw("Stored response log.",
					"reqLogID", reqLogID.String())
			}
		}()

		return nil
	}
}

func (svc *Service) SetActiveProjectID(id ulid.ULID) {
	svc.activeProjectID = id
}

func (svc *Service) ActiveProjectID() ulid.ULID {
	return svc.activeProjectID
}

func (svc *Service) SetFindReqsFilter(filter FindRequestsFilter) {
	svc.findReqsFilter = filter
}

func (svc *Service) FindReqsFilter() FindRequestsFilter {
	return svc.findReqsFilter
}

func (svc *Service) SetBypassOutOfScopeRequests(bypass bool) {
	svc.bypassOutOfScopeRequests = bypass
}

func (svc *Service) BypassOutOfScopeRequests() bool {
	return svc.bypassOutOfScopeRequests
}

// readBodyForLogging reads a body for logging, capped at maxBodySize bytes. It
// returns the bytes to log (truncated if needed), whether truncation occurred,
// and a reader that yields the FULL original body so the proxy can still
// forward it intact. A maxBodySize <= 0 means no limit.
func readBodyForLogging(body io.ReadCloser, maxBodySize int64) (logged []byte, truncated bool, full io.Reader, err error) {
	if maxBodySize <= 0 {
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, false, nil, err
		}

		return data, false, bytes.NewReader(data), nil
	}

	// Read one extra byte to detect whether the body exceeds the limit.
	data, err := io.ReadAll(io.LimitReader(body, maxBodySize+1))
	if err != nil {
		return nil, false, nil, err
	}

	if int64(len(data)) > maxBodySize {
		// Forward the full body: the prefix we already read plus the remainder.
		return data[:maxBodySize], true, io.MultiReader(bytes.NewReader(data), body), nil
	}

	return data, false, bytes.NewReader(data), nil
}

func ParseHTTPResponse(res *http.Response) (ResponseLog, error) {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ResponseLog{}, fmt.Errorf("reqlog: could not read body: %w", err)
	}

	return ResponseLog{
		Proto:      res.Proto,
		StatusCode: res.StatusCode,
		Status:     res.Status,
		Header:     res.Header,
		Body:       body,
	}, nil
}
