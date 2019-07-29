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

func ExampleGenerateGolang_enums() {
	const input = `
@url =  person
name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
gender = "male,female,others" (E)
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
	// import "encoding/json"
	//
	// type Person struct {
	// 	Name   string           `json:"name"`
	// 	Gender PersonGenderEnum `json:"gender"`
	// }
	//
	// func NewPerson() (Person, error) {
	// 	const value = "{\"name\": \"UNKNOWN\"}"
	// 	var object Person
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }
	//
	// type PersonGenderEnum uint8
	//
	// const (
	// 	PersonGenderMale PersonGenderEnum = iota
	// 	PersonGenderFemale
	// 	PersonGenderOthers
	// )
	//
	// func (e PersonGenderEnum) String() string {
	// 	switch e {
	// 	case PersonGenderMale:
	// 		return "male"
	// 	case PersonGenderFemale:
	// 		return "female"
	// 	case PersonGenderOthers:
	// 		return "others"
	// 	}
	// }
}

func ExampleGenerateGolang_ip() {
	const input = `
@url =  network
name = "UNKNOWN" (S)    #official name of the package, there can be no overlap (can be dot notation)
ip = "172.0.0.1" (ipaddr)
net = "2001:db8::/32" (iprange)
addresses = (Lipaddr)
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
	// 	"net"
	// )
	//
	// type Network struct {
	// 	Name      string         `json:"name"`
	// 	Ip        net.IP         `json:"ip"`
	// 	Net       schema.IPRange `json:"net"`
	// 	Addresses []net.IP       `json:"addresses"`
	// }
	//
	// func NewNetwork() (Network, error) {
	// 	const value = "{\"name\": \"UNKNOWN\", \"ip\": \"172.0.0.1\", \"net\": \"2001:db8::/32\"}"
	// 	var object Network
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }
}

func ExampleGenerateGolang_dict() {
	const input = `
@url =  parent
name = (S)    #official name of the package, there can be no overlap (can be dot notation)
data = (dictO) ! child # dict of children
tags = (M) # dict with no defined object type

@url = child
name = (S)
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
	// import "encoding/json"
	//
	// type Parent struct {
	// 	Name string                 `json:"name"`
	// 	Data map[string]Child       `json:"data"`
	// 	Tags map[string]interface{} `json:"tags"`
	// }
	//
	// func NewParent() (Parent, error) {
	// 	const value = "{}"
	// 	var object Parent
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }
	//
	// type Child struct {
	// 	Name string `json:"name"`
	// }
	//
	// func NewChild() (Child, error) {
	// 	const value = "{}"
	// 	var object Child
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }

}
