package schema

import (
	"os"
	"strings"
)

func ExampleGenerateGolang() {
	const input = `
@url =  jumpscale.digitalme.package
name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
enable = true (B)
numerics = (LN)
args = (LO) !jumpscale.digitalme.package.arg
loaders= (LO) !jumpscale.digitalme.package.loader

@url =  jumpscale.digitalme.package.arg
key = "" (S)
val =  "" (S)

@url =  jumpscale.digitalme.package.loader
giturl =  (S)
dest =  (S)
enable = true (B)
creation = (D)
	`

	schema, err := New(strings.NewReader(input))
	if err != nil {
		panic(err)
	}

	if err := GenerateGolang(os.Stdout, "test", schema); err != nil {
		panic(err)
	}

	// Output:
	// package test
	//
	// import (
	// 	"encoding/json"
	// 	schema "github.com/threefoldtech/zosv2/modules/schema"
	// )
	//
	// type JumpscaleDigitalmePackage struct {
	// 	Name     string                            `json:"name"`
	// 	Enable   bool                              `json:"enable"`
	// 	Numerics []schema.Numeric                  `json:"numerics"`
	// 	Args     []JumpscaleDigitalmePackageArg    `json:"args"`
	// 	Loaders  []JumpscaleDigitalmePackageLoader `json:"loaders"`
	// }
	//
	// func NewJumpscaleDigitalmePackage() (JumpscaleDigitalmePackage, error) {
	// 	const value = "{\"name\": \"UNKNOWN\", \"enable\": true}"
	// 	var object JumpscaleDigitalmePackage
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }
	//
	// type JumpscaleDigitalmePackageArg struct {
	// 	Key string `json:"key"`
	// 	Val string `json:"val"`
	// }
	//
	// func NewJumpscaleDigitalmePackageArg() (JumpscaleDigitalmePackageArg, error) {
	// 	const value = "{\"key\": \"\", \"val\": \"\"}"
	// 	var object JumpscaleDigitalmePackageArg
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }
	//
	// type JumpscaleDigitalmePackageLoader struct {
	// 	Giturl   string      `json:"giturl"`
	// 	Dest     string      `json:"dest"`
	// 	Enable   bool        `json:"enable"`
	// 	Creation schema.Date `json:"creation"`
	// }
	//
	// func NewJumpscaleDigitalmePackageLoader() (JumpscaleDigitalmePackageLoader, error) {
	// 	const value = "{\"enable\": true}"
	// 	var object JumpscaleDigitalmePackageLoader
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }
}
