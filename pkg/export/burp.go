package export

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"time"
)

type burpItems struct {
	XMLName     xml.Name   `xml:"items"`
	BurpVersion string     `xml:"burpVersion,attr"`
	ExportTime  string     `xml:"exportTime,attr"`
	Items       []burpItem `xml:"item"`
}

type burpItem struct {
	Time           string   `xml:"time"`
	URL            string   `xml:"url"`
	Host           burpHost `xml:"host"`
	Port           string   `xml:"port"`
	Protocol       string   `xml:"protocol"`
	Method         string   `xml:"method"`
	Path           string   `xml:"path"`
	Extension      string   `xml:"extension"`
	Request        burpData `xml:"request"`
	Status         string   `xml:"status"`
	ResponseLength int      `xml:"responselength"`
	MimeType       string   `xml:"mimetype"`
	Response       burpData `xml:"response"`
	Comment        string   `xml:"comment"`
}

type burpHost struct {
	IP    string `xml:"ip,attr"`
	Value string `xml:",chardata"`
}

type burpData struct {
	Base64 bool   `xml:"base64,attr"`
	Value  string `xml:",chardata"`
}

func defaultPort(u interface{ Port() string }, scheme string) string {
	if p := u.Port(); p != "" {
		return p
	}

	if scheme == "https" {
		return "443"
	}

	return "80"
}

// ExportBurpXML renders entries as a Burp Suite-compatible XML export. Request
// and response bytes are base64-encoded, as Burp expects.
func ExportBurpXML(entries []Entry) ([]byte, error) {
	doc := burpItems{
		BurpVersion: "2022.1",
		ExportTime:  time.Now().Format(time.RFC1123),
	}

	for _, e := range entries {
		item := burpItem{
			Time:   time.Now().Format(time.RFC1123),
			Method: e.Method,
			Request: burpData{
				Base64: true,
				Value:  base64.StdEncoding.EncodeToString(e.rawRequestBytes()),
			},
		}

		if e.URL != nil {
			item.URL = e.URL.String()
			item.Host = burpHost{Value: e.URL.Hostname()}
			item.Protocol = e.URL.Scheme
			item.Port = defaultPort(e.URL, e.URL.Scheme)
			item.Path = e.URL.RequestURI()
		}

		if e.Response != nil {
			item.Status = fmt.Sprintf("%d", e.Response.StatusCode)
			item.ResponseLength = len(e.Response.Body)
			item.MimeType = e.Response.Header.Get("Content-Type")
			item.Response = burpData{
				Base64: true,
				Value:  base64.StdEncoding.EncodeToString(e.rawResponseBytes()),
			}
		}

		doc.Items = append(doc.Items, item)
	}

	body, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("export: marshal burp xml: %w", err)
	}

	return append([]byte(xml.Header), body...), nil
}
