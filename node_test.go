package jsonquery

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestLoadURLStatus(t *testing.T) {
	for _, status := range []int{200, 201, 202, 206, 299, 300, 302, 304, 400, 401, 403, 404, 429, 500, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				fmt.Fprint(w, `{"message":"response"}`)
			}))
			defer server.Close()

			doc, err := LoadURL(server.URL)
			if status >= 200 && status < 300 {
				if err != nil {
					t.Fatal(err)
				}
				if got := doc.SelectElement("message"); got == nil || got.Value() != "response" {
					t.Fatalf("unexpected document: %#v", doc)
				}
				return
			}
			if doc != nil || err == nil {
				t.Fatalf("LoadURL = (%v, %v), want nil document and HTTP error", doc, err)
			}
			if !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("error %q does not identify HTTP status %d", err, status)
			}
		})
	}
}

func TestLoadURLRedirectStatus(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/redirect" {
					http.Redirect(w, r, "/document", http.StatusFound)
					return
				}
				w.WriteHeader(status)
				fmt.Fprint(w, `{"message":"redirected"}`)
			}))
			defer server.Close()

			doc, err := LoadURL(server.URL + "/redirect")
			if status == http.StatusOK {
				if err != nil {
					t.Fatal(err)
				}
				if got := doc.SelectElement("message"); got == nil || got.Value() != "redirected" {
					t.Fatalf("unexpected redirected document: %#v", doc)
				}
			} else if doc != nil || err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("LoadURL = (%v, %v), want final HTTP status %d", doc, err, status)
			}
		})
	}
}

type loadURLTransport func(*http.Request) (*http.Response, error)

func (f loadURLTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type loadURLBody struct {
	io.Reader
	read, closed bool
}

func (b *loadURLBody) Read(p []byte) (int, error) {
	b.read = true
	return b.Reader.Read(p)
}

func (b *loadURLBody) Close() error {
	b.closed = true
	return nil
}

func TestLoadURLClosesBody(t *testing.T) {
	client := http.DefaultClient
	defer func() { http.DefaultClient = client }()
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"success", http.StatusOK, `{"message":"response"}`, false},
		{"invalid JSON", http.StatusOK, `invalid`, true},
		{"empty success", http.StatusNoContent, ``, true},
		{"client error", http.StatusNotFound, `{"message":"missing"}`, true},
		{"server error", http.StatusServiceUnavailable, `invalid`, true},
		{"protocol upgrade", http.StatusSwitchingProtocols, `{"message":"response"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &loadURLBody{Reader: strings.NewReader(tc.body)}
			http.DefaultClient = &http.Client{Transport: loadURLTransport(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tc.status,
					Status:     fmt.Sprintf("%d %s", tc.status, http.StatusText(tc.status)),
					Header:     make(http.Header),
					Body:       body,
					Request:    r,
				}, nil
			})}

			doc, err := LoadURL("http://example.test/document")
			if (err != nil) != tc.wantErr {
				t.Fatalf("LoadURL = (%v, %v), want error: %v", doc, err, tc.wantErr)
			}
			if !body.closed {
				t.Error("response body was not closed")
			}
			if (tc.status < 200 || tc.status >= 300) && body.read {
				t.Error("HTTP error body was read")
			}
		})
	}
}

func parseString(s string) (*Node, error) {
	return Parse(strings.NewReader(s))
}

func TestParseJsonNumberArray(t *testing.T) {
	s := `[1,2,3,4,5,6]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}

	var values []float64
	for _, n := range doc.ChildNodes() {
		values = append(values, n.Value().(float64))
	}

	expected := []float64{1, 2, 3, 4, 5, 6}

	if p1, p2 := len(values), len(expected); p1 != p2 {
		t.Fatalf("got %d elements but expected %d", p1, p2)
	}
	if !reflect.DeepEqual(values, expected) {
		t.Fatalf("got %v but expected %v", values, expected)
	}
}

func TestParseJsonObject(t *testing.T) {
	s := `{
		"name":"John",
		"age":31,
		"female":false
	}`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}

	m := make(map[string]interface{})

	for _, n := range doc.ChildNodes() {
		m[n.Data] = n.Value()
	}
	expected := []struct {
		name  string
		value interface{}
	}{
		{"name", "John"},
		{"age", float64(31)},
		{"female", false},
	}
	for _, v := range expected {
		if e, g := v.value, m[v.name]; e != g {
			t.Fatalf("expected %s = %v(%T),but %s = %v(%t)", v.name, e, e, v.name, g, g)
		}
	}
}

func TestTextNodeValuePreservesJSONType(t *testing.T) {
	doc, err := parseString(`{"count": 12, "enabled": true, "name": "sensor"}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		value interface{}
		text  string
	}{
		{"count", float64(12), "12"},
		{"enabled", true, "true"},
		{"name", "sensor", "sensor"},
	} {
		parent := doc.SelectElement(tc.name)
		if parent == nil || parent.FirstChild == nil || parent.FirstChild.Type != TextNode {
			t.Fatalf("missing text node for %s", tc.name)
		}
		child := parent.FirstChild
		if got := child.Value(); !reflect.DeepEqual(got, tc.value) {
			t.Errorf("%s text value = %v (%T), want %v (%T)", tc.name, got, got, tc.value, tc.value)
		}
		if child.Data != tc.text || parent.InnerText() != tc.text {
			t.Errorf("%s rendered text changed: data=%q inner=%q", tc.name, child.Data, parent.InnerText())
		}
	}
}

func TestParseJsonObjectArray(t *testing.T) {
	s := `[
		{"models":[ "Fiesta", "Focus", "Mustang" ] }
	]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}

	first := doc.FirstChild
	models := first.SelectElement("models")

	if expected := reflect.ValueOf(models.Value()).Kind(); expected != reflect.Slice {
		t.Fatalf("expected models is slice(Array) but got %v", expected)
	}

	expected := []string{"Fiesta", "Focus", "Mustang"}
	var values []string
	for _, v := range models.Value().([]interface{}) {
		values = append(values, v.(string))
	}

	if !reflect.DeepEqual(expected, values) {
		t.Fatalf("expected %v but got %v", expected, values)
	}
}

func TestParseJson(t *testing.T) {
	s := `{
		"name":"John",
		"age":30,
		"cars": [
			{ "name":"Ford", "models":[ "Fiesta", "Focus", "Mustang" ] },
			{ "name":"BMW", "models":[ "320", "X3", "X5" ] },
			{ "name":"Fiat", "models":[ "500", "Panda" ] }
		]
	 }`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	n := doc.SelectElement("name")
	if n == nil {
		t.Fatal("n is nil")
	}
	if n.NextSibling != nil {
		t.Fatal("next sibling shoud be nil")
	}
	if e, g := "John", n.InnerText(); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	cars := doc.SelectElement("cars")
	if e, g := 3, len(cars.ChildNodes()); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
}

func TestLargeFloat(t *testing.T) {
	s := `{
		"large_number": 365823929453
	 }`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	n := doc.SelectElement("large_number")
	if n.Value() != float64(365823929453) {
		t.Fatalf("expected %v but %v", "365823929453", n.InnerText())
	}
}

func TestOutputXMLEscapesText(t *testing.T) {
	doc, err := parseString(`{"a": "x < y && y > z", "b": ["<i>", "AT&T"]}`)
	if err != nil {
		t.Fatal(err)
	}
	out := doc.OutputXML()
	want := `<?xml version="1.0" encoding="utf-8"?><root><a>x &lt; y &amp;&amp; y &gt; z</a><b>&lt;i&gt;</b><b>AT&amp;T</b></root>`
	if out != want {
		t.Fatalf("OutputXML() = %s, want %s", out, want)
	}

	// The output must be well-formed and decode back to the original text.
	var v struct {
		A string   `xml:"a"`
		B []string `xml:"b"`
	}
	if err := xml.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("OutputXML() is not well-formed XML: %v", err)
	}
	if v.A != "x < y && y > z" || len(v.B) != 2 || v.B[0] != "<i>" || v.B[1] != "AT&T" {
		t.Fatalf("decoded %+v", v)
	}
}
