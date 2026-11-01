package jsonquery

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestNullTextUsesJSONSpelling(t *testing.T) {
	doc, err := Parse(strings.NewReader(`{"v":null,"s":"null","f":false,"n":0}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, want string }{{"/v", "null"}, {"/s", "null"}, {"/f", "false"}, {"/n", "0"}} {
		n := FindOne(doc, tc.path)
		if n == nil {
			t.Fatal(tc.path)
		}
		if got := n.InnerText(); got != tc.want {
			t.Errorf("%s text=%q want%q", tc.path, got, tc.want)
		}
	}
	d := xml.NewDecoder(strings.NewReader(FindOne(doc, "/v").OutputXML()))
	var decoded strings.Builder
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if x, ok := tok.(xml.CharData); ok {
			decoded.Write([]byte(x))
		}
	}
	if decoded.String() != "null" {
		t.Errorf("decoded null=%q", decoded.String())
	}
	if FindOne(doc, "/v").Value() != nil {
		t.Fatal("null value type changed")
	}
}
