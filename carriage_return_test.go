package jsonquery

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestOutputXMLPreservesCarriageReturns(t *testing.T) {
	doc, err := Parse(strings.NewReader(`{"v":"a\rb\r\nc\td\ne & < >"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := "a\rb\r\nc\td\ne & < >"
	var got strings.Builder
	d := xml.NewDecoder(strings.NewReader(doc.OutputXML()))
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if x, ok := tok.(xml.CharData); ok {
			got.Write([]byte(x))
		}
	}
	if got.String() != want {
		t.Errorf("decoded text=%q, want%q", got.String(), want)
	}
}
