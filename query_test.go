package jsonquery

import (
	"strings"
	"testing"

	"github.com/antchfx/xpath"
)

func BenchmarkSelectorCache(b *testing.B) {
	DisableSelectorCache = false
	for i := 0; i < b.N; i++ {
		getQuery("/AAA/BBB/DDD/CCC/EEE/ancestor::*")
	}
}

func BenchmarkDisableSelectorCache(b *testing.B) {
	DisableSelectorCache = true
	for i := 0; i < b.N; i++ {
		getQuery("/AAA/BBB/DDD/CCC/EEE/ancestor::*")
	}
}

func TestNavigator(t *testing.T) {
	s := `{
		"name":"John",
		"age":30,
		"cars": [
			{ "name":"Ford", "models":[ "Fiesta", "Focus", "Mustang" ] },
			{ "name":"BMW", "models":[ "320", "X3", "X5" ] },
			{ "name":"Fiat", "models":[ "500", "Panda" ] }
		]
	 }`
	doc, _ := parseString(s)

	nav := CreateXPathNavigator(doc)
	nav.MoveToRoot()
	if nav.NodeType() != xpath.RootNode {
		t.Fatal("node type is not RootNode")
	}
	// Move to first child(age).
	if e, g := true, nav.MoveToChild(); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	if e, g := "age", nav.Current().Data; e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	if e, g := float64(30), nav.GetValue().(float64); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	// Move to next sibling node(cars).
	if e, g := true, nav.MoveToNext(); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	if e, g := "cars", nav.Current().Data; e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	m := make(map[string][]string)
	// Move to cars child node.
	cur := nav.Copy()
	for ok := nav.MoveToChild(); ok; ok = nav.MoveToNext() {
		// Move to <element> node.
		// <element><models>...</models><name>Ford</name></element>
		cur1 := nav.Copy()
		var name string
		var models []string
		// name || models
		for ok := nav.MoveToChild(); ok; ok = nav.MoveToNext() {
			cur2 := nav.Copy()
			n := nav.Current()
			if n.Data == "name" {
				name = n.InnerText()
			} else {
				for ok := nav.MoveToChild(); ok; ok = nav.MoveToNext() {
					cur3 := nav.Copy()
					models = append(models, nav.Value())
					nav.MoveTo(cur3)
				}
			}
			nav.MoveTo(cur2)
		}
		nav.MoveTo(cur1)
		m[name] = models
	}

	nav.MoveTo(cur)
	// move to name.
	if e, g := true, nav.MoveToNext(); e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	// move to cars
	nav.MoveToPrevious()
	if e, g := "cars", nav.Current().Data; e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	// move to age.
	nav.MoveToFirst()
	if e, g := "age", nav.Current().Data; e != g {
		t.Fatalf("expected %v but %v", e, g)
	}
	nav.MoveToParent()
	if g := nav.Current().Type; g != DocumentNode {
		t.Fatalf("node type is not DocumentNode")
	}
}

func TestToXML(t *testing.T) {
	s := `{
	"name":"John",
	"age":31, 
	"female":false 
  }`
	doc, _ := Parse(strings.NewReader(s))
	expected := `<?xml version="1.0" encoding="utf-8"?><root><age>31</age><female>false</female><name>John</name></root>`
	if got := doc.OutputXML(); got != expected {
		t.Fatalf("expected %s, but got %s", expected, got)
	}
}

func TestArrayToXML(t *testing.T) {
	s := `[1,2,3,4]`
	doc, _ := Parse(strings.NewReader(s))
	expected := `<?xml version="1.0" encoding="utf-8"?><root><1>1</1><2>2</2><3>3</3><4>4</4></root>`
	if got := doc.OutputXML(); got != expected {
		t.Fatalf("expected %s, but got %s", expected, got)
	}
}

func TestNestToArray(t *testing.T) {
	s := `{
		"address": {
		  "city": "Nara",
		  "postalCode": "630-0192",
		  "streetAddress": "naist street"
		},
		"age": 26,
		"name": "John",
		"phoneNumbers": [
		  {
			"number": "0123-4567-8888",
			"type": "iPhone"
		  },
		  {
			"number": "0123-4567-8910",
			"type": "home"
		  }
		]
	  }`
	doc, _ := Parse(strings.NewReader(s))
	expected := `<?xml version="1.0" encoding="utf-8"?><root><address><city>Nara</city><postalCode>630-0192</postalCode><streetAddress>naist street</streetAddress></address><age>26</age><name>John</name><phoneNumbers><number>0123-4567-8888</number><type>iPhone</type></phoneNumbers><phoneNumbers><number>0123-4567-8910</number><type>home</type></phoneNumbers></root>`
	if got := doc.OutputXML(); got != expected {
		t.Fatalf("expected \n%s, but got \n%s", expected, got)
	}
}

func TestQuery(t *testing.T) {
	doc, err := Parse(strings.NewReader(BooksExample))
	if err != nil {
		t.Fatal(err)
	}
	q := "/store/bicycle"
	n := FindOne(doc, q)
	if n == nil {
		t.Fatal("should matched 1 but got nil")
	}
	q = "/store/bicycle/color"
	n = FindOne(doc, q)
	if n == nil {
		t.Fatal("should matched 1 but got nil")
	}
	if n.Data != "color" {
		t.Fatalf("expected data is color, but got %s", n.Data)
	}
}

func TestQueryWhere(t *testing.T) {
	doc, err := Parse(strings.NewReader(BooksExample))
	if err != nil {
		t.Fatal(err)
	}

	// for number
	q := "//*[price<=12.99]"
	list := Find(doc, q)

	if got, expected := len(list), 3; got != expected {
		t.Fatalf("%s expected %d objects, but got %d", q, expected, got)
	}

	// for string
	q = "//*/isbn[text()='0-553-21311-3']"
	if n := FindOne(doc, q); n == nil {
		t.Fatal("should matched 1 but got nil")
	} else if n.Data != "isbn" {
		t.Fatalf("should matched `isbm` but got %s", n.Data)
	}
}

var BooksExample string = `{
	"store": {
	  "book": [
		{
		  "category": "reference",
		  "author": "Nigel Rees",
		  "title": "Sayings of the Century",
		  "price": 8.95
		},
		{
		  "category": "fiction",
		  "author": "Evelyn Waugh",
		  "title": "Sword of Honour",
		  "price": 12.99
		},
		{
		  "category": "fiction",
		  "author": "Herman Melville",
		  "title": "Moby Dick",
		  "isbn": "0-553-21311-3",
		  "price": 8.99
		},
		{
		  "category": "fiction",
		  "author": "J. R. R. Tolkien",
		  "title": "The Lord of the Rings",
		  "isbn": "0-395-19395-8",
		  "price": 22.99
		}
	  ],
	  "bicycle": {
		"color": "red",
		"price": 19.95
	  }
	},
	"expensive": 10
  }
`
