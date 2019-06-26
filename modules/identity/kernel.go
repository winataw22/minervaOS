package identity

import (
	"io/ioutil"
	"strings"
)

type param struct {
	key   string
	value string
}

func (p param) Key() string {
	return p.key
}

func (p param) Value() string {
	return p.value
}

type params []param

func (p params) Get(key string) string {
	for _, param := range p {
		if param.Key() == key {
			return param.value
		}
	}
	return ""
}

func (p params) Exist(key string) bool {
	for _, param := range p {
		if param.Key() == key {
			return true
		}
	}
	return false
}

func GetFarmID() (string, error) {
	params, err := readKernelParams()
	if err != nil {
		return "", err
	}

	return params.Get("farmer_id"), nil
}

func readKernelParams() (params, error) {
	b, err := ioutil.ReadFile("/proc/cmdline")
	if err != nil {
		return nil, err
	}

	params := []param{}

	items := strings.Split(string(b), " ")
	for _, item := range items {
		ss := strings.SplitN(item, "=", 2)
		param := param{}
		if len(ss) == 2 {
			param.key = ss[0]
			param.value = ss[1]
		} else {
			param.value = strings.TrimSpace(item)
		}

		params = append(params, param)
	}
	return params, nil
}
