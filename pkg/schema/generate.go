package schema

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
)

var (
	// this is only the list of the types implemented so far
	// in go, more types can be later added to schema modules
	// and then mapped here to support it.
	goKindMap = map[Kind][2]string{
		StringKind:    {"", "string"},
		EmailKind:     {"", "string"},
		IntegerKind:   {"", "int64"},
		FloatKind:     {"", "float64"},
		BoolKind:      {"", "bool"},
		BytesKind:     {"", "[]byte"},
		DateKind:      {"github.com/threefoldtech/zos/pkg/schema", "Date"},
		DateTimeKind:  {"github.com/threefoldtech/zos/pkg/schema", "Date"},
		NumericKind:   {"github.com/threefoldtech/zos/pkg/schema", "Numeric"},
		IPAddressKind: {"net", "IP"},
		IPRangeKind:   {"github.com/threefoldtech/zos/pkg/schema", "IPRange"},
		//TODO add other types here (for example, Email, Phone, etc..)
	}
)

// GenerateGolang generate type stubs for Go from schema
func GenerateGolang(w io.Writer, pkg string, schema Schema) error {
	g := goGenerator{enums: make(map[string]Type)}
	return g.Generate(w, pkg, schema)
}

// just a namespace for generation methods
type goGenerator struct {
	//enums: because enums types are defined on the property itself
	//and not elsewere (unlike object types for example) they need
	//to added dynamically to this map while generating code, then
	//processed later, to generate the corresponding type
	enums map[string]Type
}

func (g *goGenerator) nameFromURL(u string) (string, error) {
	p, err := url.Parse(u)

	if err != nil {
		return "", err
	}
	u = strings.ReplaceAll(p.Path, "/", "_")
	u = strings.ReplaceAll(u, ".", "_")
	return strcase.ToCamel(u), nil
}

func (g *goGenerator) Generate(w io.Writer, pkg string, schema Schema) error {
	j := jen.NewFile(pkg)

	for _, obj := range schema {
		if err := g.object(j, obj); err != nil {
			return err
		}

		j.Line()

	}

	for name, typ := range g.enums {
		if err := g.enum(j, name, &typ); err != nil {
			return err
		}

		j.Line()
	}

	return j.Render(w)
}

func (g *goGenerator) enumValues(typ *Type) []string {
	var values string
	if err := json.Unmarshal([]byte(typ.Default), &values); err != nil {
		panic(fmt.Errorf("failed to parse enum values: `%s`: %s", typ.Default, err))
	}
	return strings.Split(values, ",")
}

func (g *goGenerator) enum(j *jen.File, name string, typ *Type) error {
	typName := fmt.Sprintf("%sEnum", name)
	j.Type().Id(typName).Id("uint8").Line()
	j.Const().DefsFunc(func(group *jen.Group) {
		for i, value := range g.enumValues(typ) {
			n := fmt.Sprintf("%s%s", name, strcase.ToCamel(strings.TrimSpace(value)))
			if i == 0 {
				group.Id(n).Id(typName).Op("=").Iota()
			} else {
				group.Id(n)
			}
		}
	}).Line()

	j.Func().Params(jen.Id("e").Id(typName)).Id("String").Params().Id("string").BlockFunc(func(group *jen.Group) {
		group.Switch(jen.Id("e")).BlockFunc(func(group *jen.Group) {
			for _, value := range g.enumValues(typ) {
				value = strings.TrimSpace(value)
				n := fmt.Sprintf("%s%s", name, strcase.ToCamel(value))
				group.Case(jen.Id(n)).Block(jen.Return(jen.Lit(value)))
			}
		})

		group.Return(jen.Lit("UNKNOWN"))
	}).Line()

	return nil
}

func (g *goGenerator) renderType(typ *Type) (jen.Code, error) {
	if typ == nil {
		return jen.Interface(), nil
	}

	switch typ.Kind {
	case DictKind:
		// Dicts in jumpscale do not provide an "element" type
		// In that case (if element is not provided) a map of interface{} is
		// used. If an element is provided, this element will be used instead
		elem, err := g.renderType(typ.Element)
		if err != nil {
			return nil, err
		}
		return jen.Map(jen.Id("string")).Add(elem), nil
	case ListKind:
		elem, err := g.renderType(typ.Element)
		if err != nil {
			return nil, err
		}
		return jen.Op("[]").Add(elem), nil
	case ObjectKind:
		// we assume that object name is defined later in the file
		o, err := g.nameFromURL(typ.Reference)
		if err != nil {
			return nil, err
		}
		return jen.Qual("", o), nil
	default:
		m, ok := goKindMap[typ.Kind]
		if !ok {
			return nil, errors.Errorf("unsupported type in the go generator: %s", typ.Kind)
		}

		return jen.Qual(m[0], m[1]), nil
	}
}

func (g *goGenerator) object(j *jen.File, obj *Object) error {
	structName, err := g.nameFromURL(obj.URL)
	if err != nil {
		return err
	}
	var structErr error
	j.Type().Id(structName).StructFunc(func(group *jen.Group) {
		for _, prop := range obj.Properties {
			name := strcase.ToCamel(prop.Name)

			var typ jen.Code
			if prop.Type.Kind == EnumKind {
				// enums are handled differently than any other
				// type because they are not defined elsewere.
				// except on the property itself. so we need
				// to generate a totally new type for it.
				// TODO: do we need to have a fqn for the enum ObjectPropertyEnum instead?
				enumName := structName + name
				enumTypName := fmt.Sprintf("%sEnum", enumName)
				g.enums[enumName] = prop.Type
				typ = jen.Id(enumTypName)
			} else {
				typ, err = g.renderType(&prop.Type)
				if err != nil {
					structErr = errors.Wrapf(err, "object(%s).property(%s)", obj.URL, name)
					return
				}
			}

			group.Id(name).Add(typ).Tag(map[string]string{"json": prop.Name})
		}
	})

	// add the new Method!
	j.Func().Id(fmt.Sprintf("New%s", structName)).Params().Params(jen.Id(structName), jen.Id("error")).BlockFunc(func(group *jen.Group) {
		group.Const().Id("value").Op("=").Lit(obj.Default())
		group.Var().Id("object").Id(structName)

		group.If(jen.Id("err").Op(":=").Qual("encoding/json", "Unmarshal").Call(
			jen.Op("[]").Id("byte").Parens(jen.Id("value")),
			jen.Op("&").Id("object"),
		), jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Id("object"), jen.Id("err")),
		)

		group.Return(jen.Id("object"), jen.Nil())
	})

	return structErr
}
