package kernel

import (
	"fmt"
	"testing"
)

func validateKeyValue(mapping Params, key string, value string, t *testing.T) {
	if val, ok := mapping[key]; ok {
		if val[0] != value {
			t.Fatal(fmt.Printf("Value %s != %s", val, value))
		}
	} else {
		t.Fatal(fmt.Printf("Key %s missing", key))
	}

}

func validateMultiValueForKey(opts Params, key string, values []string, t *testing.T) {
	if vals, ok := opts[key]; ok {
		if len(vals) != len(values) {
			t.Fatalf("no multiple values for %s", key)
		}
		for i, v := range vals {
			t.Logf("got value %+v", v)
			if v != values[i] {
				t.Fatalf("%s index %d is not eq to %s", values[i], i, v)
			}
		}
	}
	t.Logf("%+v", opts)
}

func TestCmdParsing(t *testing.T) {
	params := parseParams("zerotier=mynetwork")
	validateKeyValue(params, "zerotier", "mynetwork", t)

	params = parseParams("noautonic=enp2s0f0 noautonic=enp2s0f1")
	vals := []string{"enp2s0f0", "enp2s0f1"}
	validateMultiValueForKey(params, "noautonic", vals, t)

	params = parseParams(`something   zerotier="my network"  development`)
	validateKeyValue(params, "zerotier", "my network", t)

	if !params.Exists("something") {
		t.Error("`something` is not set")
	}

	if !params.Exists("development") {
		t.Error("`development` is not set")
	}

	if values, ok := params.Get("zerotier"); ok {
		if len(values) != 1 {
			t.Error("zerotier values should be of length 1")
		}
		if values[0] != "my network" {
			t.Error("zerotier value is wrong")
		}
	} else {
		t.Error("`zerotier` is not set")
	}
}
