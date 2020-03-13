package schema

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirective(t *testing.T) {
	dir, err := directive("@url = hello world # comment")

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Directive{Key: "url", Value: "hello world"}, dir); !ok {
		t.Error()
	}

	dir, err = directive("@url hello world # comment")

	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestProperty(t *testing.T) {
	prop, err := property(`something* = "hello world" (S) # comment`)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Property{
		Name:    "something",
		Indexed: true,
		Type: Type{
			Default: `"hello world"`,
			Kind:    StringKind,
		},
	}, prop); !ok {
		t.Error()
	}

	prop, err = property("wrong hello world # comment")

	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestPropertyEnum(t *testing.T) {
	prop, err := property(`currency = "EUR,USD,TFT,AED,GBP" (E) # comment`)

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, Property{
		Name:    "currency",
		Indexed: false,
		Type: Type{
			Default: `"EUR,USD,TFT,AED,GBP"`,
			Kind:    EnumKind,
		},
	}, prop); !ok {
		t.Error()
	}

	prop, err = property("wrong hello world # comment")

	if ok := assert.Error(t, err); !ok {
		t.Fatal()
	}
}

func TestTypeDef(t *testing.T) {
	cases := []struct {
		Input  string
		Output *Type
	}{
		{
			`"hello world" (S)`,
			&Type{
				Default: `"hello world"`,
				Kind:    StringKind,
			},
		},
		{
			`(LO) ! reference.to.another.object`,
			&Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "reference.to.another.object",
				},
			},
		},
		{
			`(Lmultiline) ! reference.to.another.object`,
			&Type{
				Kind: ListKind,
				Element: &Type{
					Kind: MultilineKind,
				},
			},
		},
		{
			`"1,2,3" (LI)`,
			&Type{
				Kind:    ListKind,
				Default: `"1,2,3"`,
				Element: &Type{
					Kind: IntegerKind,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.Input, func(t *testing.T) {
			typ, err := typeDef(c.Input)

			if ok := assert.NoError(t, err); !ok {
				t.Fatal()
			}

			if ok := assert.EqualValues(t, c.Output, typ); !ok {
				t.Error()
			}
		})
	}

	failures := []string{
		"(i)", "(Iunknown)", `"default" (S) garbage`,
	}

	for _, c := range failures {
		_, err := typeDef(c)

		if ok := assert.Error(t, err); !ok {
			t.Error()
		}
	}
}

func TestTokenScan(t *testing.T) {
	cases := []struct {
		Input  string
		Output []string
	}{
		{"I", []string{"I"}},
		{"II", []string{"I", "I"}},
		{"Lstring", []string{"L", "string"}},
		{"LstringI", []string{"L", "string", "I"}},
		{"LO", []string{"L", "O"}},
		{"", nil},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("case.%s", c.Input), func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(c.Input))
			scanner.Split(tokenSplit)

			var out []string
			for scanner.Scan() {
				out = append(out, scanner.Text())
			}

			if ok := assert.Equal(t, c.Output, out); !ok {
				t.Error()
			}
		})
	}

}

func TestSimpleNew(t *testing.T) {
	const input = `
# this is a comment

@url =  jumpscale.digitalme.package
name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
enable = true (B)
args = (LO) !jumpscale.digitalme.package.arg
loaders= (LO) !jumpscale.digitalme.package.loader
	`

	schema, err := New(strings.NewReader(input))

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, schema, 1); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, &Object{
		URL:    "jumpscale.digitalme.package",
		IsRoot: true,
		Properties: []Property{
			{Name: "name", Type: Type{
				Default: `"UNKNOWN"`,
				Kind:    StringKind,
			}},
			{Name: "enable", Type: Type{
				Default: `true`,
				Kind:    BoolKind,
			}},
			{Name: "args", Type: Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "jumpscale.digitalme.package.arg",
				},
			}},
			{Name: "loaders", Type: Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "jumpscale.digitalme.package.loader",
				},
			}},
		},
	}, schema[0]); !ok {
		t.Error()
	}
}

func TestNew(t *testing.T) {
	input := `
	@url =  jumpscale.digitalme.package
	name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
	enable = true (B)
	args = (LO) !jumpscale.digitalme.package.arg
	loaders= (LO) !jumpscale.digitalme.package.loader

	@url =  jumpscale.digitalme.package.arg
	key = "" (S)
	val =  "" (S)

	@url =  jumpscale.digitalme.package.loader
	giturl =  (S)
	dest =  (S)
	enable = true (B)
	`

	schema, err := New(strings.NewReader(input))

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, schema, 3); !ok {
		t.Fatal()
	}

	if ok := assert.Equal(t, &Object{
		URL:    "jumpscale.digitalme.package",
		IsRoot: true,
		Properties: []Property{
			{Name: "name", Type: Type{
				Default: `"UNKNOWN"`,
				Kind:    StringKind,
			}},
			{Name: "enable", Type: Type{
				Default: `true`,
				Kind:    BoolKind,
			}},
			{Name: "args", Type: Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "jumpscale.digitalme.package.arg",
				},
			}},
			{Name: "loaders", Type: Type{
				Kind: ListKind,
				Element: &Type{
					Kind:      ObjectKind,
					Reference: "jumpscale.digitalme.package.loader",
				},
			}},
		},
	}, schema[0]); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, &Object{
		URL: "jumpscale.digitalme.package.arg",
		Properties: []Property{
			{Name: "key", Type: Type{
				Default: `""`,
				Kind:    StringKind,
			}},
			{Name: "val", Type: Type{
				Default: `""`,
				Kind:    StringKind,
			}},
		},
	}, schema[1]); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, &Object{
		URL: "jumpscale.digitalme.package.loader",
		Properties: []Property{
			{Name: "giturl", Type: Type{
				Kind: StringKind,
			}},
			{Name: "dest", Type: Type{
				Kind: StringKind,
			}},
			{Name: "enable", Type: Type{
				Default: "true",
				Kind:    BoolKind,
			}},
		},
	}, schema[2]); !ok {
		t.Error()
	}
}

func TestObjectDefault(t *testing.T) {
	const input = `
@url =  jumpscale.digitalme.package
name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
enable = true (B)
unset = (S)
args = "1,2,3" (LI)
int = 1 (I)
strings = ["hello", "world"] (LS)
`

	schema, err := New(strings.NewReader(input))

	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, schema, 1); !ok {
		t.Fatal()
	}

	obj := schema[0]

	type T struct {
		Name    string   `json:"name"`
		Enable  bool     `json:"enable"`
		Args    []int64  `json:"args"`
		Int     int64    `json:"int"`
		Strings []string `json:"strings"`
	}

	var data T
	if err := json.Unmarshal([]byte(obj.Default()), &data); err != nil {
		t.Fatal(err)
	}

	if ok := assert.Equal(t, T{
		Name:    "UNKNOWN",
		Enable:  true,
		Args:    []int64{1, 2, 3},
		Int:     1,
		Strings: []string{"hello", "world"},
	}, data); !ok {
		t.Error()
	}
}
