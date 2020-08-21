package jsonquery

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path"
	"sort"
	"strings"
	"testing"
)

func parseString(s string) (*Node, error) {
	return Parse(strings.NewReader(s))
}

func TestParseJsonNumberArray(t *testing.T) {
	s := `[1,2,3,4,5,6]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	// output like below:
	// <element>1</element>
	// <element>2</element>
	// ...
	// <element>6</element>
	if e, g := 6, len(doc.ChildNodes()); e != g {
		t.Fatalf("excepted %v but got %v", e, g)
	}
	var v []string
	for _, n := range doc.ChildNodes() {
		v = append(v, n.InnerText())
	}
	if got, expected := strings.Join(v, ","), "1,2,3,4,5,6"; got != expected {
		t.Fatalf("got %v but expected %v", got, expected)
	}
}

func TestParseJsonObject(t *testing.T) {
	s := `{
		"name":"John",
		"age":31,
		"city":"New York"
	}`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	// output like below:
	// <name>John</name>
	// <age>31</age>
	// <city>New York</city>
	m := make(map[string]string)
	for _, n := range doc.ChildNodes() {
		m[n.Data] = n.InnerText()
	}
	expected := []struct {
		name, value string
	}{
		{"name", "John"},
		{"age", "31"},
		{"city", "New York"},
	}
	for _, v := range expected {
		if e, g := v.value, m[v.name]; e != g {
			t.Fatalf("expected %v=%v,but %v=%v", v.name, e, v.name, g)
		}
	}
}

func TestParseJsonObjectArray(t *testing.T) {
	s := `[
		{ "name":"Ford", "models":[ "Fiesta", "Focus", "Mustang" ] },
		{ "name":"BMW", "models":[ "320", "X3", "X5" ] },
        { "name":"Fiat", "models":[ "500", "Panda" ] }
	]`
	doc, err := parseString(s)
	if err != nil {
		t.Fatal(err)
	}
	/**
	<element>
		<name>Ford</name>
		<models>
			<element>Fiesta</element>
			<element>Focus</element>
			<element>Mustang</element>
		</models>
	</element>
	<element>
		<name>BMW</name>
		<models>
			<element>320</element>
			<element>X3</element>
			<element>X5</element>
		</models>
	</element>
	....
	*/
	if e, g := 3, len(doc.ChildNodes()); e != g {
		t.Fatalf("expected %v, but %v", e, g)
	}
	m := make(map[string][]string)
	for _, n := range doc.ChildNodes() {
		// Go to the next of the element list.
		var name string
		var models []string
		for _, e := range n.ChildNodes() {
			if e.Data == "name" {
				// a name node.
				name = e.InnerText()
			} else {
				// a models node.
				for _, k := range e.ChildNodes() {
					models = append(models, k.InnerText())
				}
			}
		}
		// Sort models list.
		sort.Strings(models)
		m[name] = models

	}
	expected := []struct {
		name, value string
	}{
		{"Ford", "Fiesta,Focus,Mustang"},
		{"BMW", "320,X3,X5"},
		{"Fiat", "500,Panda"},
	}
	for _, v := range expected {
		if e, g := v.value, strings.Join(m[v.name], ","); e != g {
			t.Fatalf("expected %v=%v,but %v=%v", v.name, e, v.name, g)
		}
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
	if n.InnerText() != "365823929453" {
		t.Fatalf("expected %v but %v", "365823929453", n.InnerText())
	}
}

func TestJSON(t *testing.T) {
	files := []string{
		"basic.json",
		"screen_v3_01.json",
		"screen_v3_02.json",
		"screen_v3_03.json",
		"screen_v3_04.json",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			originalBytes, err := ioutil.ReadFile(path.Join("testdata", file))
			if err != nil {
				t.Fatal(err)
			}

			var iOriginalJSON interface{}
			err = json.NewDecoder(bytes.NewReader(originalBytes)).Decode(&iOriginalJSON)
			if err != nil {
				t.Fatal(err)
			}

			doc, err := Parse(bytes.NewReader(originalBytes))
			if err != nil {
				t.Fatal(err)
			}

			iDocJSON, err := doc.JSON()
			if err != nil {
				t.Fatal(err)
			}

			originalJSONBytes, err := json.Marshal(iOriginalJSON)
			if err != nil {
				t.Fatal(err)
			}
			docJSONBytes, err := json.Marshal(iDocJSON)
			if err != nil {
				t.Fatal(err)
			}
			if string(originalJSONBytes) != string(docJSONBytes) {
				t.Fatalf(
					"JSON from doc is different from original JSON \nOriginal: %s \nDoc:      %s",
					string(originalJSONBytes),
					string(docJSONBytes),
				)
			}
		})
	}
}

func TestFindAssetIDs(t *testing.T) {
	originalBytes, err := ioutil.ReadFile(path.Join("testdata", "screen_v3_01.json"))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := Parse(bytes.NewReader(originalBytes))
	if err != nil {
		t.Fatal(err)
	}

	strAssetIDs := []string{"4629", "4627", "4631", "4630"}
	nodes := Find(doc, "/layers//exportOptions//asset_id")
	if n := len(nodes); n != 4 {
		t.Fatalf("Expected 4 nodes but got only %v", n)
	}
	for i, n := range nodes {
		if n.Data != "asset_id" {
			t.Fatalf("Expected asset_id but got %s", n.Data)
		}
		if n.InnerText() != strAssetIDs[i] {
			t.Fatalf("Expected %s but got %s", strAssetIDs[i], n.InnerText())
		}
	}
}