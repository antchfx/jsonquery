package jsonquery

import (
	"strings"
	"testing"

	"github.com/antchfx/xpath"
)

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
	/**
	<age>30</age>
	<cars>
		<element>
			<models>...</models>
			<name>Ford</name>
		</element>
		<element>
			<models>...</models>
			<name>BMW</name>
		</element>
		<element>
			<models>...</models>
			<name>Fiat</name>
		</element>
	</cars>
	<name>John</name>
	*/
	nav := CreateXPathNavigator(doc)
	nav.MoveToRoot()
	if nav.NodeType() != xpath.RootNode {
		t.Fatal("node type is not RootNode")
	}
	// Move to first child(age).
	if e, g := true, nav.MoveToChild(); e != g {
		t.Fatalf("expected %s but %s", e, g)
	}
	if e, g := "age", nav.Current().Data; e != g {
		t.Fatalf("expected %s but %s", e, g)
	}
	if e, g := "30", nav.Value(); e != g {
		t.Fatalf("expected %s but %s", e, g)
	}
	// Move to next sibling node(cars).
	if e, g := true, nav.MoveToNext(); e != g {
		t.Fatalf("expected %b but %b", e, g)
	}
	if e, g := "cars", nav.Current().Data; e != g {
		t.Fatalf("expected %s but %s", e, g)
	}
	m := make(map[string][]string)
	// Move to cars child node.
	cur := nav.Copy()
	for ok := nav.MoveToChild(); ok; ok = nav.MoveToNext() {
		// Move to <element> node.
		// <element><models>...</models><name>Ford</name></element>
		cur_1 := nav.Copy()
		var name string
		var models []string
		// name || models
		for ok := nav.MoveToChild(); ok; ok = nav.MoveToNext() {
			cur_2 := nav.Copy()
			n := nav.Current()
			if n.Data == "name" {
				name = n.InnerText()
			} else {
				for ok := nav.MoveToChild(); ok; ok = nav.MoveToNext() {
					cur_3 := nav.Copy()
					models = append(models, nav.Value())
					nav.MoveTo(cur_3)
				}
			}
			nav.MoveTo(cur_2)
		}
		nav.MoveTo(cur_1)
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
			t.Fatalf("expected %s=%s,but %s=%s", v.name, e, v.name, g)
		}
	}
	nav.MoveTo(cur)
	// move to name.
	if e, g := true, nav.MoveToNext(); e != g {
		t.Fatalf("expected %b but %b", e, g)
	}
	// move to cars
	nav.MoveToPrevious()
	if e, g := "cars", nav.Current().Data; e != g {
		t.Fatalf("expected %s but %s", e, g)
	}
	// move to age.
	nav.MoveToFirst()
	if e, g := "age", nav.Current().Data; e != g {
		t.Fatalf("expected %s but %s", e, g)
	}
	nav.MoveToParent()
	if g := nav.Current().Type; g != DocumentNode {
		t.Fatalf("node type is not DocumentNode")
	}
}
