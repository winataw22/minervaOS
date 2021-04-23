package gridtypes

import (
	"crypto/ed25519"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrWorkloadNotFound error
	ErrWorkloadNotFound = fmt.Errorf("workload not found")
)

// https://github.com/threefoldtech/vgrid/blob/main/zosreq/req_deployment_root.v

// WorkloadGetter is used to get a workload by name inside
// the deployment context. Mainly used to validate dependency
type WorkloadGetter interface {
	Get(name string) (*WorkloadWithID, error)
}

// WorkloadByTypeGetter is used to get a list of workloads
// of specific type from a workload container like a deployment
type WorkloadByTypeGetter interface {
	ByType(typ WorkloadType) []*WorkloadWithID
}

// KeyGetter interface to get key by twin ids
type KeyGetter interface {
	GetKey(twin uint32) (ed25519.PublicKey, error)
}

// Deployment structure
type Deployment struct {
	Version              int                  `json:"version"`
	TwinID               uint32               `json:"twin_id"`
	DeploymentID         uint32               `json:"deployment_id"`
	Metadata             string               `json:"metadata"`
	Description          string               `json:"description"`
	Expiration           Timestamp            `json:"expiration"`
	SignatureRequirement SignatureRequirement `json:"signature_requirement"`
	Workloads            []Workload           `json:"workloads"`
}

// WorkloadWithID wrapper around workload type
// that holds the global workload ID
type WorkloadWithID struct {
	*Workload
	ID WorkloadID
}

// SignatureRequest struct
type SignatureRequest struct {
	TwinID   uint32 `json:"twin_id"`
	Required bool   `json:"required"`
	Weight   uint   `json:"weight"`
}

// Challenge computes challenge for SignatureRequest
func (r *SignatureRequest) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", r.TwinID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%t", r.Required); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", r.Weight); err != nil {
		return err
	}

	return nil
}

// Signature struct
type Signature struct {
	TwinID    uint32 `json:"twin_id"`
	Signature string `json:"signature"`
}

// SignatureRequirement struct
type SignatureRequirement struct {
	Requests       []SignatureRequest `json:"requests"`
	WeightRequired uint               `json:"weight_required"`
	Signatures     []Signature        `json:"signatures"`
}

// Challenge computes challenge for SignatureRequest
func (r *SignatureRequirement) Challenge(w io.Writer) error {
	for _, request := range r.Requests {
		if err := request.Challenge(w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%d", r.WeightRequired); err != nil {
		return err
	}

	return nil
}

// ChallengeHash computes the hash of the challenge signed
// by the user. used for validation
func (d *Deployment) ChallengeHash() ([]byte, error) {
	hash := md5.New()
	if err := d.Challenge(hash); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// Challenge computes challenge for SignatureRequest
func (d *Deployment) Challenge(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "%d", d.Version); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.TwinID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.DeploymentID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", d.Metadata); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s", d.Description); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%d", d.Expiration); err != nil {
		return err
	}

	for _, wl := range d.Workloads {
		if err := wl.Challenge(w); err != nil {
			return err
		}
	}

	return nil
}

// Valid validates deployment structure
func (d *Deployment) Valid() error {
	names := make(map[string]struct{})
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		name := wl.Name
		if err := IsValidName(name); err != nil {
			return errors.Wrapf(err, "name '%s' is invalid", name)
		}

		if _, ok := names[name]; ok {
			return fmt.Errorf("multiple workloads with the same name '%s'", name)
		}
		names[name] = struct{}{}

		if err := wl.Valid(d); err != nil {
			return err
		}
	}

	return nil
}

// Sign adds a signature to deployment given twin id
func (d *Deployment) Sign(twin uint32, sk ed25519.PrivateKey) error {
	message, err := d.ChallengeHash()
	if err != nil {
		return err
	}
	signatureBytes := ed25519.Sign(sk, message)
	signature := hex.EncodeToString(signatureBytes)
	for i := range d.SignatureRequirement.Signatures {
		sig := &d.SignatureRequirement.Signatures[i]
		// update
		if sig.TwinID == twin {
			sig.Signature = signature
			return nil
		}
	}

	d.SignatureRequirement.Signatures = append(
		d.SignatureRequirement.Signatures, Signature{
			TwinID:    twin,
			Signature: signature,
		})
	return nil
}

// Verify verifies user signature
func (d *Deployment) Verify(getter KeyGetter) error {
	message, err := d.ChallengeHash()
	if err != nil {
		return err
	}

	requirements := &d.SignatureRequirement
	get := func(twin uint32) (Signature, bool) {
		for _, sig := range requirements.Signatures {
			if sig.TwinID == twin {
				return sig, true
			}
		}
		return Signature{}, false
	}

	originatorFound := false
	var weight uint
	for _, request := range requirements.Requests {
		if request.TwinID == d.TwinID {
			originatorFound = true
			request.Required = true // we force that this is a required signature
		}

		signature, ok := get(request.TwinID)
		if !ok && request.Required {
			return fmt.Errorf("missing required signature for twin '%d'", request.TwinID)
		}

		pk, err := getter.GetKey(request.TwinID)
		if err != nil {
			return errors.Wrapf(err, "failed to get public key for twin '%d'", request.TwinID)
		}

		bytes, err := hex.DecodeString(signature.Signature)
		if err != nil {
			return errors.Wrap(err, "invalid signature")
		}

		if !ed25519.Verify(pk, message, bytes) {
			return fmt.Errorf("failed to verify signature")
		}
		weight += request.Weight
	}

	if !originatorFound {
		return fmt.Errorf("originator twin id must be in the signature requests")
	}

	if weight < requirements.WeightRequired {
		return fmt.Errorf("required signature weight is not reached")
	}

	return nil
}

// Get a workload by name
func (d *Deployment) Get(name string) (*WorkloadWithID, error) {
	if err := IsValidName(name); err != nil {
		return nil, err
	}

	for i := range d.Workloads {
		wl := &d.Workloads[i]
		if wl.Name == name {
			id, _ := NewWorkloadID(d.TwinID, d.DeploymentID, name)
			return &WorkloadWithID{
				Workload: wl,
				ID:       id,
			}, nil
		}
	}

	return nil, errors.Wrapf(ErrWorkloadNotFound, "no workload with name '%s'", name)
}

// GetType gets a reservation by name if only of the correct type.
func (d *Deployment) GetType(name string, typ WorkloadType) (*WorkloadWithID, error) {
	wl, err := d.Get(name)
	if err != nil {
		return nil, err
	}
	if wl.Type != typ {
		return nil, fmt.Errorf("workload '%s' of wrong type", name)
	}

	return wl, nil
}

// ByType gets all workloads from this reservation by type.
func (d *Deployment) ByType(typ WorkloadType) []*WorkloadWithID {
	var results []*WorkloadWithID
	for i := range d.Workloads {
		wl := &d.Workloads[i]
		if wl.Type == typ {
			id, err := NewWorkloadID(d.TwinID, d.DeploymentID, wl.Name)
			if err != nil {
				log.Warn().Err(err).Msg("deployment has invalid name. please run validation")
				continue
			}

			results = append(
				results,
				&WorkloadWithID{
					Workload: wl,
					ID:       id,
				},
			)
		}
	}

	return results
}

// Upgrade validates n as an updated version of d, and return an Upgrade description
// for the steps that the node needs to take.
func (d *Deployment) Upgrade(n *Deployment) (*Upgrade, error) {
	if err := n.Valid(); err != nil {
		return nil, errors.Wrap(err, "new deployment is invalid")
	}

	if d.TwinID != n.TwinID || d.DeploymentID != n.DeploymentID {
		return nil, fmt.Errorf("cannot change deployment or twin id")
	}

	expected := d.Version + 1
	if expected != n.Version {
		return nil, fmt.Errorf("expecting deployment version %d, got %d", expected, n.Version)
	}

	current := make(map[string]*Workload)
	for i := range d.Workloads {
		wl := &d.Workloads[i]

		current[wl.Name] = wl
	}

	update := make([]*WorkloadWithID, 0)
	add := make([]*WorkloadWithID, 0)
	remove := make([]*WorkloadWithID, 0)

	for i := range n.Workloads {
		l := &n.Workloads[i]
		id, err := NewWorkloadID(n.TwinID, n.DeploymentID, l.Name)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get build workload ID")
		}
		wl := &WorkloadWithID{
			Workload: l,
			ID:       id,
		}
		older, ok := current[wl.Name]

		if !ok {
			if wl.Version == expected {
				// newly added workload
				add = append(add, wl)
			} else {
				return nil, fmt.Errorf("invalid version number for workload '%s' expected '%d'", wl.Name, expected)
			}
		} else {
			// modifying a type.
			if older.Type != wl.Type {
				return nil, fmt.Errorf("cannot change workload type '%s'", wl.Name)
			}
			// update the state of new deployment
			wl.Result = older.Result
			// older version that exists.
			// so
			if wl.Version == expected {
				// should added to 'update' pile
				update = append(update, wl)
			}
			// other wise. we leave it untouched
		}

		// in both cases, we remove this from the current list
		delete(current, wl.Name)
	}

	for _, wl := range current {
		id, _ := NewWorkloadID(d.TwinID, d.DeploymentID, wl.Name)

		remove = append(remove, &WorkloadWithID{
			Workload: wl,
			ID:       id,
		})
	}

	return &Upgrade{ToAdd: add, ToUpdate: update, ToRemove: remove}, nil
}

// Upgrade procedure structure
type Upgrade struct {
	ToAdd    []*WorkloadWithID
	ToUpdate []*WorkloadWithID
	ToRemove []*WorkloadWithID
}
