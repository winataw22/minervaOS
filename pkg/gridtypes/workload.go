package gridtypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// WorkloadType type
type WorkloadType string

var (
	// workloadTypes with built in known types
	workloadTypes = map[WorkloadType]WorkloadData{}
)

// RegisterType register a new workload type
func RegisterType(t WorkloadType, d WorkloadData) {
	if reflect.TypeOf(d).Kind() != reflect.Struct {
		panic("only structures are supported")
	}
	if _, ok := workloadTypes[t]; ok {
		panic("type already registered")
	}

	workloadTypes[t] = d
}

//Types return a list of all registered types
func Types() []WorkloadType {
	types := make([]WorkloadType, 0, len(workloadTypes))
	for typ := range workloadTypes {
		types = append(types, typ)
	}

	return types
}

// Valid checks if this is a known reservation type
func (t WorkloadType) Valid() error {
	if _, ok := workloadTypes[t]; !ok {
		return fmt.Errorf("invalid reservation type")
	}

	return nil
}

func (t WorkloadType) String() string {
	return string(t)
}

//Capacity the expected capacity of a workload
type Capacity struct {
	CRU   uint64 `json:"cru"`
	SRU   uint64 `json:"sru"`
	HRU   uint64 `json:"hru"`
	MRU   uint64 `json:"mru"`
	IPV4U uint64 `json:"ipv4u"`
}

// Zero returns true if capacity is zero
func (c *Capacity) Zero() bool {
	return c.CRU == 0 && c.SRU == 0 && c.HRU == 0 && c.MRU == 0 && c.IPV4U == 0
}

// Add increments value of capacity with o
func (c *Capacity) Add(o *Capacity) {
	c.CRU += o.CRU
	c.MRU += o.MRU
	c.SRU += o.SRU
	c.HRU += o.HRU
	c.IPV4U += o.IPV4U
}

// WorkloadData interface
type WorkloadData interface {
	Valid(getter WorkloadGetter) error
	Challenge(io.Writer) error
	Capacity() (Capacity, error)
}

// MustMarshal is a utility function to quickly serialize workload data
func MustMarshal(data WorkloadData) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return json.RawMessage(bytes)
}

// Workload struct
type Workload struct {
	//Version is version of reservation object
	Version int `json:"version"`
	//Name is unique workload name per deployment  (required)
	Name string `json:"name"`
	// Type of the reservation (container, zdb, vm, etc...)
	Type WorkloadType `json:"type"`
	// Data is the reservation type arguments.
	Data json.RawMessage `json:"data"`
	// Date of creation
	Created Timestamp `json:"created"`
	// Metadata is custom user metadata
	Metadata string `json:"metadata"`
	//Description
	Description string `json:"description"`
	// Result of reservation
	Result Result `json:"result"`
}

// WorkloadData loads data of workload into WorkloadData object
func (w *Workload) WorkloadData() (WorkloadData, error) {
	if err := w.Type.Valid(); err != nil {
		return nil, err
	}

	typ := workloadTypes[w.Type] // this returns a copy
	value := reflect.New(reflect.TypeOf(typ)).Interface()

	if err := json.Unmarshal(w.Data, &value); err != nil {
		return nil, errors.Wrapf(err, "failed to load data into object of type '%T'", value)
	}

	return value.(WorkloadData), nil
}

// IsResult returns true if workload has a valid result of the given state
func (w *Workload) IsResult(state ResultState) bool {
	return !w.Result.IsNil() && w.Result.State == state
}

// Valid validate reservation
func (w *Workload) Valid(getter WorkloadGetter) error {
	if err := IsValidName(w.Name); err != nil {
		return errors.Wrap(err, "invalid workload name")
	}

	if err := w.Type.Valid(); err != nil {
		return err
	}

	data, err := w.WorkloadData()
	if err != nil {
		return err
	}

	return data.Valid(getter)
}

//Challenge implementation
func (w *Workload) Challenge(i io.Writer) error {
	data, err := w.WorkloadData()
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(i, "%d", w.Version); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(i, "%s", w.Type.String()); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(i, "%s", w.Metadata); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(i, "%s", w.Description); err != nil {
		return err
	}

	return data.Challenge(i)
}

// Capacity returns the used capacity by this workload
func (w *Workload) Capacity() (Capacity, error) {
	data, err := w.WorkloadData()
	if err != nil {
		return Capacity{}, err
	}

	return data.Capacity()
}

// //Sign signs the signature given the private key
// func (w *Workload) Sign(sk ed25519.PrivateKey) error {
// 	if len(sk) != ed25519.PrivateKeySize {
// 		return fmt.Errorf("invalid secure key")
// 	}

// 	var buf bytes.Buffer
// 	if err := w.Challenge(&buf); err != nil {
// 		return errors.Wrap(err, "failed to create the signature challenge")
// 	}

// 	w.Signature = hex.EncodeToString(ed25519.Sign(sk, buf.Bytes()))

// 	return nil
// }

// AppendTag appends tags
func AppendTag(t, n Tag) Tag {
	if t == nil {
		t = Tag{}
	}

	for k, v := range n {
		t[k] = v
	}

	return t
}

// Tag is custom tag to mark certain reservations
type Tag map[string]string

func (t Tag) String() string {
	var builder strings.Builder
	for k, v := range t {
		if builder.Len() != 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(k)
		builder.WriteString(": ")
		builder.WriteString(v)
	}

	return builder.String()
}

// ResultState type
type ResultState string

const (
	// StateError constant
	StateError = "error"
	// StateOk constant
	StateOk = "ok"
	//StateDeleted constant
	StateDeleted = "deleted"
)

// Result is the struct filled by the node
// after a reservation object has been processed
type Result struct {
	// Time when the result is sent
	Created Timestamp `json:"created"`
	// State of the deployment (ok,error)
	State ResultState `json:"state"`
	// if State is "error", then this field contains the error
	// otherwise it's nil
	Error string `json:"message"`
	// Data is the information generated by the provisioning of the workload
	// its type depend on the reservation type
	Data json.RawMessage `json:"data"`
}

// IsNil checks if Result is the zero values
func (r *Result) IsNil() bool {
	// ideally this should be implemented like this
	// emptyResult := Result{}
	// return reflect.DeepEqual(r, &emptyResult)
	//
	// but unfortunately, the empty Result coming from the explorer already have some fields set
	// (like the type)
	// so instead we gonna check the Data and the Created filed

	return r.Created == 0 && (len(r.Data) == 0 || bytes.Equal(r.Data, nullRaw))
}

var (
	//emptyResult is the Result zero value
	nullRaw, _ = json.Marshal(nil)
)

// Bytes returns a slice of bytes container all the information
// used to sign the Result object
func (r *Result) Bytes() ([]byte, error) {
	buf := &bytes.Buffer{}
	if _, err := buf.WriteString(string(r.State)); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString(r.Error); err != nil {
		return nil, err
	}
	if _, err := buf.Write(r.Data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
