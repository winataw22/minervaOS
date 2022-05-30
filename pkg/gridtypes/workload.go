package gridtypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/pkg/errors"
)

// WorkloadType type
type WorkloadType string

var (
	// workloadTypes with built in known types
	workloadTypes = map[WorkloadType]WorkloadData{}
	// sharableWorkloadTypes are workload types that can be
	// accessed from other deployments (as read only)
	// but only modifiable from deployments that creates it.
	sharableWorkloadTypes = map[WorkloadType]struct{}{}
)

// RegisterType register a new workload type. This is used by zos to "declare"
// the workload types it supports. please check `zos` sub package for all
// supported types.
// Note: a user never need to call this, it's done by zos libraries.
func RegisterType(t WorkloadType, d WorkloadData) {
	if reflect.TypeOf(d).Kind() != reflect.Struct {
		panic("only structures are supported")
	}
	if _, ok := workloadTypes[t]; ok {
		panic("type already registered")
	}

	workloadTypes[t] = d
}

// RegisterSharableType same as RegisterType, but also register
// this type as sharable, which means this type can be accessed (referenced)
// from other deploments. But only modifiable from the type deployment
// that created it.
func RegisterSharableType(t WorkloadType, d WorkloadData) {
	RegisterType(t, d)
	sharableWorkloadTypes[t] = struct{}{}
}

//Types return a list of all registered types
func Types() []WorkloadType {
	types := make([]WorkloadType, 0, len(workloadTypes))
	for typ := range workloadTypes {
		types = append(types, typ)
	}

	return types
}

func IsSharable(typ WorkloadType) bool {
	_, ok := sharableWorkloadTypes[typ]
	return ok
}

// Valid checks if this is a known reservation type
func (t WorkloadType) Valid() error {
	if _, ok := workloadTypes[t]; !ok {
		return fmt.Errorf("invalid reservation type '%s'", t.String())
	}

	return nil
}

func (t WorkloadType) String() string {
	return string(t)
}

//Capacity the expected capacity of a workload
type Capacity struct {
	CRU   uint64 `json:"cru"`
	SRU   Unit   `json:"sru"`
	HRU   Unit   `json:"hru"`
	MRU   Unit   `json:"mru"`
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
	// Version is version of reservation object. On deployment creation, version must be 0
	// then only workloads that need to be updated must match the version of the deployment object.
	// if a deployment update message is sent to a node it does the following:
	// - validate deployment version
	// - check workloads list, if a version is not matching the new deployment version, the workload is untouched
	// - if a workload version is same as deployment, the workload is "updated"
	// - if a workload is removed, the workload is deleted.
	Version uint32 `json:"version"`
	//Name is unique workload name per deployment  (required)
	Name Name `json:"name"`
	// Type of the reservation (container, zdb, vm, etc...)
	Type WorkloadType `json:"type"`
	// Data is the reservation type arguments.
	Data json.RawMessage `json:"data"`
	// Metadata is user specific meta attached to deployment, can be used to link this
	// deployment to other external systems for automation
	Metadata string `json:"metadata"`
	//Description human readale description of the workload
	Description string `json:"description"`
	// Result of reservation, set by the node
	Result Result `json:"result"`
}

func (w Workload) WithResults(result Result) Workload {
	w.Result = result
	return w
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

// ResultState type
type ResultState string

func (s ResultState) IsAny(state ...ResultState) bool {
	for _, in := range state {
		if s == in {
			return true
		}
	}

	return false
}

func (s ResultState) IsOkay() bool {
	return s.IsAny(StateOk, StatePaused)
}

const (
	// StateInit is the first state of the workload on storage
	StateInit ResultState = "init"
	// StateUnChanged is a special error state it means there was an error
	// running the action, but this error did not break previous state.
	StateUnChanged ResultState = "unchanged"
	// StateError constant
	StateError ResultState = "error"
	// StateOk constant
	StateOk ResultState = "ok"
	// StateDeleted constant
	StateDeleted ResultState = "deleted"
	// StatePaused constant
	StatePaused ResultState = "paused"
)

var (
	validStates = []ResultState{
		StateInit, StateUnChanged, StateError, StateOk, StateDeleted, StatePaused,
	}
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

func (r *Result) Valid() error {
	if r.Created == 0 {
		return fmt.Errorf("created time must be set")
	}

	in := func() bool {
		for _, state := range validStates {
			if r.State == state {
				return true
			}
		}
		return false
	}

	if !in() {
		return fmt.Errorf("invalid result state")
	}

	return nil
}

// Unmarshal a shortcut for json.Unmarshal
func (r *Result) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Data, v)
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

	return r.State == "" && r.Created == 0 && (len(r.Data) == 0 || bytes.Equal(r.Data, nullRaw))
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
