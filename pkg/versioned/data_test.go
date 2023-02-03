package versioned

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReader(t *testing.T) {
	// versioned object
	buf := bytes.NewBufferString(`"1.2.0" {"name": "Test", "value": "success"}`)

	reader, err := NewReader(buf)
	require.NoError(t, err)

	if ok := assert.Equal(t, MustParse("1.2.0"), reader.Version()); !ok {
		t.Fatal()
	}

	// now read the object itself
	var data struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	dec := json.NewDecoder(reader)
	err = dec.Decode(&data)
	require.NoError(t, err)

	if ok := assert.Equal(t, "Test", data.Name); !ok {
		t.Error()
	}

	if ok := assert.Equal(t, "success", data.Value); !ok {
		t.Error()
	}
}

type BadReader struct{}

func (b BadReader) Read(d []byte) (int, error) {
	return 0, fmt.Errorf("caused by io error")
}

func TestReaderInvalid(t *testing.T) {
	// case 1, no version information in stream
	buf := bytes.NewBufferString(`{"name": "Test", "value": "success"}`)

	_, err := NewReader(buf)
	require.True(t, IsNotVersioned(err))

	// case 2, invalid version string
	buf = bytes.NewBufferString(`"abc" {"name": "Test", "value": "success"}`)

	_, err = NewReader(buf)
	require.True(t, IsNotVersioned(err))

	// case 3, empty input
	buf = bytes.NewBufferString("")
	_, err = NewReader(buf)
	require.True(t, IsNotVersioned(err))

	// case 4, underlying io error
	var bad BadReader
	_, err = NewReader(bad)
	require.Error(t, err)
	require.False(t, IsNotVersioned(err))
}

func TestWriterReader(t *testing.T) {
	type Data struct {
		Name string
		Age  float64
		Tags []string
	}

	var buf bytes.Buffer

	writer, err := NewWriter(&buf, MustParse("1.0.0"))
	require.NoError(t, err)

	// Note you can replace json here with any encoder u like
	enc := json.NewEncoder(writer)
	data := Data{"Test", 20.0, []string{"version", "check"}}
	require.NoError(t, enc.Encode(data))

	// loading
	reader, err := NewReader(&buf)
	require.NoError(t, err)

	if ok := assert.Equal(t, MustParse("1.0.0"), reader.Version()); !ok {
		t.Fatal()
	}

	dec := json.NewDecoder(reader)
	var loaded Data

	require.NoError(t, dec.Decode(&loaded))
	if ok := assert.Equal(t, data, loaded); !ok {
		t.Fatal()
	}
}

func TestLoadSaveFile(t *testing.T) {
	data := make([]byte, 500)
	_, err := rand.Read(data)
	require.NoError(t, err)

	file, err := os.CreateTemp("", "test-")
	require.NoError(t, err)
	// lazy way to get a temp file path
	path := file.Name()
	defer os.Remove(path)
	file.Close()

	version := MustParse("1.2.0-beta")
	err = WriteFile(path, version, data, 0400)
	require.NoError(t, err)

	loadedVersion, loadedData, err := ReadFile(path)
	require.NoError(t, err)

	require.Equal(t, version, loadedVersion)
	require.Equal(t, data, loadedData)
}

func TestLoadNotVersioned(t *testing.T) {
	data := make([]byte, 500)
	_, err := rand.Read(data)
	require.NoError(t, err)

	file, err := os.CreateTemp("", "test-")
	require.NoError(t, err)
	// lazy way to get a temp file path
	path := file.Name()
	defer os.Remove(path)
	_, err = file.Write(data)
	require.NoError(t, err)
	file.Close()

	_, loadedData, err := ReadFile(path)
	require.Error(t, err)
	require.True(t, IsNotVersioned(err))

	require.Equal(t, data, loadedData)
}

func ExampleNewReader() {
	latest := MustParse("1.2.0")
	// 1- Open file contains data
	buf := bytes.NewBufferString(`"1.0.1" "my data goes here"`)

	// 2- create versioned reader
	reader, err := NewReader(buf)
	if err != nil {
		// no version in data, take another action!
		panic(err)
	}

	fmt.Println("data version is:", reader.Version())
	dec := json.NewDecoder(reader)

	if reader.Version().Compare(latest) <= 0 {
		//data version is older than or equal latest
		var data string
		if err := dec.Decode(&data); err != nil {
			panic(err)
		}
		fmt.Println("data is:", "my data goes here")
	}

	// Output:
	// data version is: 1.0.1
	// data is: my data goes here
}

func Example() {
	type DataV1 struct {
		FullName string `json:"name"`
	}

	type DataV2 struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	//latest := MustParse("1.2.0")
	// 1- Open file contains data
	bits := `{"name": "John Smith"}`

	buf := bytes.NewBufferString(bits)
	// 2- create versioned reader
	reader, err := NewReader(buf)
	if IsNotVersioned(err) {
		reader = NewVersionedReader(MustParse("1.0.0"), bytes.NewBufferString(bits))
	} else if err != nil {
		panic(err)
	}

	dec := json.NewDecoder(reader)
	var data DataV2 // final object
	var resave bool
	if reader.Version().EQ(MustParse("1.0.0")) { //V1 object
		// data migration from v1 to v2
		var d1 DataV1
		if err := dec.Decode(&d1); err != nil {
			panic(err)
		}
		parts := strings.SplitN(d1.FullName, " ", 2)
		data = DataV2{
			FirstName: parts[0],
			LastName:  parts[1],
		}
		resave = true
	} else if reader.Version().EQ(MustParse("2.0.0")) { //V2 (current) version
		if err := dec.Decode(&data); err != nil {
			panic(err)
		}
	} else {
		panic("unknown version")
	}

	if resave {
		var buf bytes.Buffer
		writer, _ := NewWriter(&buf, MustParse("2.0.0"))
		enc := json.NewEncoder(writer)
		if err := enc.Encode(data); err != nil {
			panic(err)
		}

		fmt.Println(buf.String())
	}

	// Output:
	// "2.0.0"{"first_name":"John","last_name":"Smith"}
}
