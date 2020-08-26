package jsonquery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
)

// A NodeType is the type of a Node.
type NodeType uint

type contentType string

const (
	// DocumentNode is a document object that, as the root of the document tree,
	// provides access to the entire XML document.
	DocumentNode NodeType = iota
	// ElementNode is an element.
	ElementNode
	// TextNode is the text content of a node.
	TextNode
)

const (
	arrayType   = contentType("array")
	objectType  = contentType("object")
	stringType  = contentType("string")
	float64Type = contentType("float64")
	boolType    = contentType("bool")
	nullType    = contentType("null")
)

// A Node consists of a NodeType and some Data (tag name for
// element nodes, content for text) and are part of a tree of Nodes.
type Node struct {
	Parent, PrevSibling, NextSibling, FirstChild, LastChild *Node

	Type NodeType
	Data string

	level       int
	contentType contentType
	idata       interface{}
	skipped     bool
}

// ChildNodes gets all child nodes of the node.
func (n *Node) ChildNodes() []*Node {
	var a []*Node
	for nn := n.FirstChild; nn != nil; nn = nn.NextSibling {
		a = append(a, nn)
	}
	return a
}

// InnerText gets the value of the node and all its child nodes.
func (n *Node) InnerText() string {
	var output func(*bytes.Buffer, *Node)
	output = func(buf *bytes.Buffer, n *Node) {
		if n.Type == TextNode {
			buf.WriteString(n.Data)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			output(buf, child)
		}
	}
	var buf bytes.Buffer
	output(&buf, n)
	return buf.String()
}

func (n *Node) InnerData() interface{} {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		return child.InnerData()
	}

	return n.idata
}

func (n *Node) SetInnerData(idata interface{}) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		child.SetInnerData(idata)
	}

	n.idata = idata
}

func (n *Node) GetParent(level int) *Node {
	if n.Parent.level == level {
		return n.Parent
	}

	return n.Parent.GetParent(level)
}

func (n *Node) Skipped() {
	n.skipped = true
}

func outputXML(buf *bytes.Buffer, n *Node) {
	switch n.Type {
	case ElementNode:
		if n.Data == "" {
			buf.WriteString("<element>")
		} else {
			buf.WriteString("<" + n.Data + ">")
		}
	case TextNode:
		buf.WriteString(n.Data)
		return
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		outputXML(buf, child)
	}
	if n.Data == "" {
		buf.WriteString("</element>")
	} else {
		buf.WriteString("</" + n.Data + ">")
	}
}

// OutputXML prints the XML string.
func (n *Node) OutputXML() string {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0"?>`)
	for n := n.FirstChild; n != nil; n = n.NextSibling {
		outputXML(&buf, n)
	}
	return buf.String()
}

func (n *Node) JSON(skipped bool) (interface{}, error) {
	switch n.contentType {
	case arrayType:
		arr := make([]interface{}, 0)
		for _, node := range n.ChildNodes() {
			if skipped && node.skipped {
				continue
			}

			value, err := node.JSON(skipped)
			if err != nil {
				return nil, err
			}
			arr = append(arr, value)
		}
		return arr, nil
	case objectType:
		obj := map[string]interface{}{}
		for _, node := range n.ChildNodes() {
			if skipped && node.skipped {
				continue
			}

			value, err := node.JSON(skipped)
			if err != nil {
				return nil, err
			}
			obj[node.Data] = value
		}
		return obj, nil
	case float64Type:
		return strconv.ParseFloat(n.InnerText(), 64)
	case stringType:
		return n.InnerText(), nil
	case boolType:
		return strconv.ParseBool(n.InnerText())
	case nullType:
		return nil, nil
	}

	return nil, fmt.Errorf("%v type is not supported", n.contentType)
}

// SelectElement finds the first of child elements with the
// specified name.
func (n *Node) SelectElement(name string) *Node {
	for nn := n.FirstChild; nn != nil; nn = nn.NextSibling {
		if nn.Data == name {
			return nn
		}
	}
	return nil
}

// LoadURL loads the JSON document from the specified URL.
func LoadURL(url string) (*Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return Parse(resp.Body)
}

func parseValue(x interface{}, top *Node, level int) {
	addNode := func(n *Node) {
		if n.level == top.level {
			top.NextSibling = n
			n.PrevSibling = top
			n.Parent = top.Parent
			if top.Parent != nil {
				top.Parent.LastChild = n
			}
		} else if n.level > top.level {
			n.Parent = top
			if top.FirstChild == nil {
				top.FirstChild = n
				top.LastChild = n
			} else {
				t := top.LastChild
				t.NextSibling = n
				n.PrevSibling = t
				top.LastChild = n
			}
		}
	}

	switch v := x.(type) {
	case []interface{}:
		top.contentType = arrayType

		for _, vv := range v {
			n := &Node{Type: ElementNode, level: level}
			addNode(n)
			parseValue(vv, n, level+1)
		}
	case map[string]interface{}:
		top.contentType = objectType

		// The Go’s map iteration order is random.
		// (https://blog.golang.org/go-maps-in-action#Iteration-order)
		var keys []string
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			n := &Node{Data: key, Type: ElementNode, level: level}
			addNode(n)
			parseValue(v[key], n, level+1)
		}
	case string:
		top.contentType = stringType

		n := &Node{Data: v, Type: TextNode, level: level, idata: v}
		addNode(n)
	case float64:
		top.contentType = float64Type

		s := strconv.FormatFloat(v, 'f', -1, 64)
		n := &Node{Data: s, Type: TextNode, level: level, idata: v}
		addNode(n)
	case bool:
		top.contentType = boolType

		s := strconv.FormatBool(v)
		n := &Node{Data: s, Type: TextNode, level: level, idata: v}
		addNode(n)
	case nil:
		top.contentType = nullType

		n := &Node{Data: "", Type: TextNode, level: level, idata: v}
		addNode(n)
	}
}

func parse(b []byte) (*Node, error) {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}

	doc := &Node{Type: DocumentNode}
	switch v.(type) {
	case []interface{}:
		doc.contentType = arrayType
	case map[string]interface{}:
		doc.contentType = objectType
	}

	parseValue(v, doc, 1)
	return doc, nil
}

// Parse JSON document.
func Parse(r io.Reader) (*Node, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parse(b)
}
