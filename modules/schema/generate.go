package schema

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/iancoleman/strcase"
)

var (
	// this is only the list of the types implemented so far
	// in go, more types can be later added to schema modules
	// and then mapped here to support it.
	goKindMap = map[Kind][2]string{
		StringKind:  {"", "string"},
		IntegerKind: {"", "int64"},
		FloatKind:   {"", "float64"},
		BoolKind:    {"", "bool"},
		DateKind:    {"github.com/threefoldtech/zosv2/modules/schema", "Date"},
		NumericKind: {"github.com/threefoldtech/zosv2/modules/schema", "Numeric"},
		//TODO add other types here (for example, Email, Phone, IP, etc..)
	}
)

// GenerateGolang generate type stubs for Go from schema
func GenerateGolang(w io.Writer, pkg string, schema Schema) error {
	g := goGenerator{}
	return g.Generate(w, pkg, schema)
}

// just a namespace for generation methods
type goGenerator struct{}

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

	return j.Render(w)
}

func (g *goGenerator) renderType(typ *Type) (jen.Code, error) {
	if typ == nil {
		return nil, fmt.Errorf("type is nil")
	}

	switch typ.Kind {

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
			return nil, fmt.Errorf("unsupported type in the go generator: %v", typ)
		}

		return jen.Qual(m[0], m[1]), nil
	}

}

func (g *goGenerator) object(j *jen.File, obj *Object) error {
	name, err := g.nameFromURL(obj.URL)
	if err != nil {
		return err
	}
	var structErr error
	j.Type().Id(name).StructFunc(func(group *jen.Group) {
		for _, prop := range obj.Properties {
			name := strcase.ToCamel(prop.Name)
			typ, err := g.renderType(&prop.Type)
			if err != nil {
				structErr = err
				return
			}

			group.Id(name).Add(typ).Tag(map[string]string{"json": prop.Name})
		}
	})

	// add the new Method!
	j.Func().Id(fmt.Sprintf("New%s", name)).Params().Params(jen.Id(name), jen.Id("error")).BlockFunc(func(group *jen.Group) {
		group.Const().Id("value").Op("=").Lit(obj.Default())
		group.Var().Id("object").Id(name)

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
