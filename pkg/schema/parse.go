package schema

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
)

const (
	// URL directive
	URL = "url"
)

var (
	directiveRe = regexp.MustCompile(`@(\w+)\s*=\s*([^\#]+)`)
	propertyRe  = regexp.MustCompile(`(\w+)(\*){0,2}\s*=\s*([^\#]+)`)
	typeRe      = regexp.MustCompile(`^([^\(]*)\(([^\)]+)\)\s*(?:!(.+))?$`)
)

// Schema is a container for all objects defined in the source
type Schema []*Object

// Object defines a schema object
type Object struct {
	URL        string
	IsRoot     bool
	Directives []Directive
	Properties []Property
}

func (o *Object) listDefault(buf *strings.Builder, typ *Type) {
	// list can be in format [o1, o2], or "o1, o2"
	// we need to figure out the type, and try to generate
	// a valid json representation of the list

	def := typ.Default
	if def[0] == '[' {
		// we assume it's a valid list
		buf.WriteString(def)
		return
	}
	// cut the " or the ' at the ends of the array
	cuttest := "\""
	if def[0] == '\'' {
		cuttest = "'"
	}
	def = strings.Trim(def, cuttest)
	buf.WriteRune('[')
	buf.WriteString(def)
	buf.WriteByte(']')
}

//Default generates a json string that holds the default value
//for the object
func (o *Object) Default() string {
	var buf strings.Builder
	buf.WriteRune('{')

	for _, prop := range o.Properties {
		if len(prop.Type.Default) == 0 || prop.Type.Kind == EnumKind {
			// if default is not set OR if enum type, we don't set a default
			// Note: we ignore enum type because enums always have defaults
			// set to 0 (first item in the enum type)
			continue
		}

		if buf.Len() > 1 { // the 1 is the {
			buf.WriteString(", ")
		}

		buf.WriteString(fmt.Sprintf(`"%s": `, prop.Name))
		if prop.Type.Kind == ListKind {
			o.listDefault(&buf, &prop.Type)
		} else {
			buf.WriteString(prop.Type.Default)
		}
	}

	buf.WriteRune('}')

	return buf.String()
}

// New reads the schema and return schema description objects
func New(r io.Reader) (schema Schema, err error) {
	scanner := bufio.NewScanner(r)
	var current *Object
	nr := 0
	for scanner.Scan() {
		nr++
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 {
			continue
		}

		switch line[0] {
		case '#':
			//comment line, we can skip it
			continue
		case '@':
			//directive line, need to be processed and acted upon
			dir, err := directive(line)
			if err != nil {
				return schema, err
			}

			if dir.Key == URL {
				//push current
				if current != nil {
					schema = append(schema, current)
				}
				current = &Object{
					URL: dir.Value,
				}
			} else if current == nil {
				//directive out of scoop!
				return schema, fmt.Errorf("unexpected directive [line %d]: %s", nr, dir)
			} else {
				current.Directives = append(current.Directives, dir)
			}
		default:
			if current == nil {
				return schema, fmt.Errorf("unexpected token [line %d]: %s", nr, line)
			}
			prop, err := property(line)
			if err != nil {
				return schema, fmt.Errorf("failed to parse property [line %d]: %s", nr, err)
			}

			current.Properties = append(current.Properties, prop)
		}
	}

	if current != nil {
		schema = append(schema, current)
	}

	// mark roots
root:
	for i := 0; i < len(schema); i++ {
		a := schema[i]
		for j := 0; j < len(schema); j++ {
			b := schema[j]
			for _, prop := range b.Properties {
				var ref string
				switch prop.Type.Kind {
				case ObjectKind:
					ref = prop.Type.Reference
				case ListKind:
					fallthrough
				case DictKind:
					if prop.Type.Element != nil {
						ref = prop.Type.Element.Reference
					}
				}

				if a.URL == ref {
					continue root
				}
			}
		}

		a.IsRoot = true
	}

	return
}

// Directive is a piece of information attached to a schema object
type Directive struct {
	Key   string
	Value string
}

func (d *Directive) String() string {
	return fmt.Sprintf("@%s = %s", d.Key, d.Value)
}

func directive(line string) (dir Directive, err error) {
	m := directiveRe.FindStringSubmatch(line)
	if m == nil {
		return dir, fmt.Errorf("invalid directive line")
	}

	return Directive{Key: strings.ToLower(m[1]), Value: strings.TrimSpace(m[2])}, nil
}

// Property defines a schema property
type Property struct {
	Name    string
	Indexed bool
	Type    Type
}

func property(line string) (property Property, err error) {
	m := propertyRe.FindStringSubmatch(line)
	if m == nil {
		return property, fmt.Errorf("invalid property definition")
	}

	property.Name = m[1]
	property.Indexed = m[2] == "*"
	typ, err := typeDef(strings.TrimSpace(m[3]))
	if err != nil {
		return property, nil
	}
	property.Type = *typ

	return
}

// Kind defines the kind of a type
type Kind int

// List of all the supported JSX types
const (
	UnknownKind Kind = iota
	IntegerKind
	FloatKind
	BoolKind
	StringKind
	MobileKind
	EmailKind
	IPPortKind
	IPAddressKind
	IPRangeKind
	DateKind
	DateTimeKind
	NumericKind
	GUIDKind
	DictKind
	ListKind
	ObjectKind
	YamlKind
	MultilineKind
	HashKind
	BytesKind
	PercentKind
	URLKind
	EnumKind
)

func (k Kind) String() string {
	for s, i := range symbols {
		if i == k {
			return s
		}
	}

	return "unknown"
}

var (
	symbols = map[string]Kind{
		// short symbols, they must be upper case
		"S": StringKind,
		"I": IntegerKind,
		"F": FloatKind,
		"B": BoolKind,
		"D": DateKind,
		"T": DateTimeKind,
		"N": NumericKind,
		"L": ListKind,
		"O": ObjectKind,
		"P": PercentKind,
		"U": URLKind,
		"H": HashKind,
		"E": EnumKind,
		"M": DictKind, // Map

		// long symbols, they must be lower case
		"str":       StringKind,
		"string":    StringKind,
		"int":       IntegerKind,
		"integer":   IntegerKind,
		"float":     FloatKind,
		"bool":      BoolKind,
		"mobile":    MobileKind,
		"email":     EmailKind,
		"ipport":    IPPortKind,
		"ipaddr":    IPAddressKind,
		"ipaddress": IPAddressKind,
		"iprange":   IPRangeKind,
		"date":      DateKind,
		"time":      DateTimeKind,
		"numeric":   NumericKind,
		"guid":      GUIDKind,
		"dict":      DictKind,
		"map":       DictKind,
		"yaml":      YamlKind,
		"multiline": MultilineKind,
		"hash":      HashKind,
		"bin":       BytesKind,
		"bytes":     BytesKind,
		"percent":   PercentKind,
		"url":       URLKind,
		"object":    ObjectKind,
		"enum":      EnumKind,
	}
)

// Type holds type information for a property
type Type struct {
	Kind      Kind
	Default   string
	Reference string
	Element   *Type
}

func tokenSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) == 0 && atEOF {
		return 0, nil, nil
	}

	if unicode.IsUpper(rune(data[0])) {
		return 1, data[0:1], nil
	}

	var i int
	for i = 0; i < len(data) && unicode.IsLower(rune(data[i])); i++ {

	}

	if i < len(data) {
		// we hit an upper case char
		return i, data[0:i], nil
	}

	// we didn't hit eof yet. it means we can still continue
	// scanning and fine more lower case letters
	if !atEOF {
		return 0, nil, nil
	}

	// otherwise we return the whole thing
	return len(data), data, nil
}

func typeDef(def string) (*Type, error) {
	// typeDef parses a type text and create a solid type definition
	// current default values is ignored!
	// <default> (Type)

	m := typeRe.FindStringSubmatch(def)
	if m == nil {
		return nil, fmt.Errorf("invalid type definition")
	}

	var root Type
	root.Default = strings.TrimSpace(m[1])
	typeStr := m[2]
	reference := strings.TrimSpace(m[3])

	scanner := bufio.NewScanner(strings.NewReader(typeStr))
	scanner.Split(tokenSplit)

	current := &root

	for scanner.Scan() {
		if current.Kind != UnknownKind {
			current.Element = new(Type)
			current = current.Element
		}

		symbol := scanner.Text()
		kind, ok := symbols[symbol]
		if !ok {
			return nil, fmt.Errorf("unknown kind: '%s'", symbol)
		}

		current.Kind = kind
		if current.Kind == ObjectKind {
			current.Reference = reference
		}

	}

	return &root, nil
}
