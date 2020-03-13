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
	// 	schema "github.com/threefoldtech/zos/pkg/schema"
	// )
	//
	// type JumpscaleDigitalmePackage struct {
	// 	ID       schema.ID                         `bson:"_id" json:"id"`
	// 	Name     string                            `bson:"name" json:"name"`
	// 	Enable   bool                              `bson:"enable" json:"enable"`
	// 	Numerics []schema.Numeric                  `bson:"numerics" json:"numerics"`
	// 	Args     []JumpscaleDigitalmePackageArg    `bson:"args" json:"args"`
	// 	Loaders  []JumpscaleDigitalmePackageLoader `bson:"loaders" json:"loaders"`
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
	// 	Key string `bson:"key" json:"key"`
	// 	Val string `bson:"val" json:"val"`
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
	// 	Giturl   string      `bson:"giturl" json:"giturl"`
	// 	Dest     string      `bson:"dest" json:"dest"`
	// 	Enable   bool        `bson:"enable" json:"enable"`
	// 	Creation schema.Date `bson:"creation" json:"creation"`
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
	// import (
	// 	"encoding/json"
	// 	schema "github.com/threefoldtech/zos/pkg/schema"
	// )
	//
	// type Person struct {
	// 	ID     schema.ID        `bson:"_id" json:"id"`
	// 	Name   string           `bson:"name" json:"name"`
	// 	Gender PersonGenderEnum `bson:"gender" json:"gender"`
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
	// 	return "UNKNOWN"
	// }

}

func ExampleGenerateGolang_enums2() {
	const input = `

@url = tfgrid.node.resource.price.1
cru = (F)

mru = (F)
hru = (F)
sru = (F)
nru = (F)
currency = "EUR,USD,TFT,AED,GBP" (E)
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
	// 	schema "github.com/threefoldtech/zos/pkg/schema"
	// )
	//
	// type TfgridNodeResourcePrice1 struct {
	// 	ID       schema.ID                            `bson:"_id" json:"id"`
	// 	Cru      float64                              `bson:"cru" json:"cru"`
	// 	Mru      float64                              `bson:"mru" json:"mru"`
	// 	Hru      float64                              `bson:"hru" json:"hru"`
	// 	Sru      float64                              `bson:"sru" json:"sru"`
	// 	Nru      float64                              `bson:"nru" json:"nru"`
	// 	Currency TfgridNodeResourcePrice1CurrencyEnum `bson:"currency" json:"currency"`
	// }
	//
	// func NewTfgridNodeResourcePrice1() (TfgridNodeResourcePrice1, error) {
	// 	const value = "{}"
	// 	var object TfgridNodeResourcePrice1
	// 	if err := json.Unmarshal([]byte(value), &object); err != nil {
	// 		return object, err
	// 	}
	// 	return object, nil
	// }
	//
	// type TfgridNodeResourcePrice1CurrencyEnum uint8
	//
	// const (
	// 	TfgridNodeResourcePrice1CurrencyEUR TfgridNodeResourcePrice1CurrencyEnum = iota
	// 	TfgridNodeResourcePrice1CurrencyUSD
	// 	TfgridNodeResourcePrice1CurrencyTFT
	// 	TfgridNodeResourcePrice1CurrencyAED
	// 	TfgridNodeResourcePrice1CurrencyGBP
	// )
	//
	// func (e TfgridNodeResourcePrice1CurrencyEnum) String() string {
	// 	switch e {
	// 	case TfgridNodeResourcePrice1CurrencyEUR:
	// 		return "EUR"
	// 	case TfgridNodeResourcePrice1CurrencyUSD:
	// 		return "USD"
	// 	case TfgridNodeResourcePrice1CurrencyTFT:
	// 		return "TFT"
	// 	case TfgridNodeResourcePrice1CurrencyAED:
	// 		return "AED"
	// 	case TfgridNodeResourcePrice1CurrencyGBP:
	// 		return "GBP"
	// 	}
	// 	return "UNKNOWN"
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
	// 	schema "github.com/threefoldtech/zos/pkg/schema"
	// 	"net"
	// )
	//
	// type Network struct {
	// 	ID        schema.ID      `bson:"_id" json:"id"`
	// 	Name      string         `bson:"name" json:"name"`
	// 	Ip        net.IP         `bson:"ip" json:"ip"`
	// 	Net       schema.IPRange `bson:"net" json:"net"`
	// 	Addresses []net.IP       `bson:"addresses" json:"addresses"`
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
	// import (
	// 	"encoding/json"
	// 	schema "github.com/threefoldtech/zos/pkg/schema"
	// )
	//
	// type Parent struct {
	// 	ID   schema.ID              `bson:"_id" json:"id"`
	// 	Name string                 `bson:"name" json:"name"`
	// 	Data map[string]Child       `bson:"data" json:"data"`
	// 	Tags map[string]interface{} `bson:"tags" json:"tags"`
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
	// 	Name string `bson:"name" json:"name"`
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
