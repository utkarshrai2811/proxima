package export

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	numericSegRe = regexp.MustCompile(`^\d+$`)
	uuidSegRe    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

type openAPIDoc struct {
	OpenAPI string                            `yaml:"openapi"`
	Info    openAPIInfo                       `yaml:"info"`
	Paths   map[string]map[string]openAPIOper `yaml:"paths"`
}

type openAPIInfo struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
}

type openAPIOper struct {
	Summary     string                     `yaml:"summary,omitempty"`
	Parameters  []openAPIParam             `yaml:"parameters,omitempty"`
	RequestBody *openAPIBody               `yaml:"requestBody,omitempty"`
	Responses   map[string]openAPIResponse `yaml:"responses"`
}

type openAPIParam struct {
	Name     string            `yaml:"name"`
	In       string            `yaml:"in"`
	Required bool              `yaml:"required"`
	Schema   map[string]string `yaml:"schema"`
}

type openAPIBody struct {
	Content map[string]openAPIMedia `yaml:"content"`
}

type openAPIMedia struct {
	Example any `yaml:"example,omitempty"`
}

type openAPIResponse struct {
	Description string                  `yaml:"description"`
	Content     map[string]openAPIMedia `yaml:"content,omitempty"`
}

// normalizePath replaces numeric and UUID segments with path parameters,
// returning the templated path and the parameter names in order.
func normalizePath(p string) (string, []string) {
	if p == "" {
		p = "/"
	}

	segs := strings.Split(p, "/")

	var params []string

	for i, s := range segs {
		if numericSegRe.MatchString(s) || uuidSegRe.MatchString(s) {
			name := "id"
			if len(params) > 0 {
				name = fmt.Sprintf("id%d", len(params)+1)
			}

			params = append(params, name)
			segs[i] = "{" + name + "}"
		}
	}

	return strings.Join(segs, "/"), params
}

func jsonExample(body []byte) any {
	var v any
	if err := json.Unmarshal(body, &v); err == nil {
		return v
	}

	return string(body)
}

// ExportOpenAPI analyses entries and produces a minimal OpenAPI 3.0 YAML doc.
func ExportOpenAPI(entries []Entry) ([]byte, error) {
	paths := map[string]map[string]openAPIOper{}

	for _, e := range entries {
		if e.URL == nil {
			continue
		}

		tmpl, params := normalizePath(e.URL.Path)
		method := strings.ToLower(e.Method)

		if paths[tmpl] == nil {
			paths[tmpl] = map[string]openAPIOper{}
		}

		if _, exists := paths[tmpl][method]; exists {
			continue // first observed request for this (method, path) wins
		}

		op := openAPIOper{
			Summary:   fmt.Sprintf("%s %s", e.Method, tmpl),
			Responses: map[string]openAPIResponse{},
		}

		for _, p := range params {
			op.Parameters = append(op.Parameters, openAPIParam{
				Name: p, In: "path", Required: true, Schema: map[string]string{"type": "string"},
			})
		}

		if len(e.Body) > 0 && hasJSONContentType(e.Header) {
			op.RequestBody = &openAPIBody{
				Content: map[string]openAPIMedia{"application/json": {Example: jsonExample(e.Body)}},
			}
		}

		if e.Response != nil {
			resp := openAPIResponse{Description: e.Response.Status}
			if resp.Description == "" {
				resp.Description = "response"
			}

			if len(e.Response.Body) > 0 && hasJSONContentType(e.Response.Header) {
				resp.Content = map[string]openAPIMedia{
					"application/json": {Example: jsonExample(e.Response.Body)},
				}
			}

			op.Responses[fmt.Sprintf("%d", e.Response.StatusCode)] = resp
		}

		if len(op.Responses) == 0 {
			op.Responses["200"] = openAPIResponse{Description: "OK"}
		}

		paths[tmpl][method] = op
	}

	doc := openAPIDoc{
		OpenAPI: "3.0.0",
		Info:    openAPIInfo{Title: "Proxima Export", Version: "1.0.0"},
		Paths:   paths,
	}

	body, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("export: marshal openapi yaml: %w", err)
	}

	return body, nil
}
