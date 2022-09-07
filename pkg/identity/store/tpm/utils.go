package tpm

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
)

type HexString string

func (h HexString) Bytes() ([]byte, error) {
	return hex.DecodeString(string(h))
}

type HashAlgorithm string
type KeyAlgorithm string

const (
	SHA1   HashAlgorithm = "sha1"
	SHA256 HashAlgorithm = "sha256"
	SHA384 HashAlgorithm = "sha384"
	SHA512 HashAlgorithm = "sha512"

	RSA KeyAlgorithm = "rsa"
)

type Address uint32

type PCRSelector map[HashAlgorithm][]int

func (p PCRSelector) String() string {
	// to make it consistent we need to
	// sort the the map keys first
	var keys []HashAlgorithm
	for hash := range p {
		keys = append(keys, hash)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	var buf strings.Builder
	for _, hash := range keys {
		if buf.Len() > 0 {
			buf.WriteRune('+')
		}
		buf.WriteString(string(hash))
		buf.WriteRune(':')
		for i, id := range p[hash] {
			if i != 0 {
				buf.WriteRune(',')
			}
			buf.WriteString(fmt.Sprint(id))
		}
	}

	return buf.String()
}

// File is a tmp file path to make it easier to pass files around
type File string

// Delete file
func (f File) Delete() error {
	return os.Remove(string(f))
}

// Read file contents
func (f File) Read() ([]byte, error) {
	return os.ReadFile(string(f))
}

type Object struct {
	public  File
	private File
}

func (o *Object) Delete() error {
	_ = o.private.Delete()
	_ = o.public.Delete()
	return nil
}

// Creates a temporary file handler
func NewFile(suffix string) File {
	name := fmt.Sprintf("%s%s", uuid.New().String(), suffix)
	return File(filepath.Join(os.TempDir(), name))
}

func tpm(ctx context.Context, name string, in io.Reader, out interface{}, arg ...string) error {
	name = fmt.Sprintf("tpm2_%s", name)

	cmd := exec.CommandContext(ctx, name, arg...)
	log.Debug().Msgf("executing command: %s", cmd.String())
	cmd.Stdin = in
	output, err := cmd.Output()
	if err, ok := err.(*exec.ExitError); ok && err != nil {
		return errors.Wrapf(err, "error while running command: (%s)", string(err.Stderr))
	} else if err != nil {
		return errors.Wrap(err, "failed to run tpm")
	}

	if out == nil {
		return nil
	}

	decoder := yaml.NewDecoder(bytes.NewBuffer(output))

	return decoder.Decode(out)
}

// IsTPMEnabled checks if TPM is accessible on this system
func IsTPMEnabled(ctx context.Context) bool {
	pcrs, err := PCRs(ctx)
	if err != nil {
		return false
	}

	return len(pcrs) > 0
}

// PersistedHandlers return a list of persisted handlers on the system
func PersistedHandlers(ctx context.Context) (handlers []Address, err error) {
	var strHandlers []string
	if err := tpm(ctx, "getcap", nil, &strHandlers, "handles-persistent"); err != nil {
		return nil, err
	}

	var addresses []Address
	for _, handler := range strHandlers {
		var u Address
		_, err := fmt.Sscanf(handler, "0x%x", &u)
		if err != nil {
			return nil, fmt.Errorf("failed to scan address(%s): %w", handler, err)
		}

		addresses = append(addresses, u)
	}

	return addresses, nil
}

// PCRs returns the available PCRs numbers as map of [hash-algorithm][]int
func PCRs(ctx context.Context) (map[string][]int, error) {
	var data struct {
		Top []map[string][]int `yaml:"selected-pcrs"`
	}

	if err := tpm(ctx, "getcap", nil, &data, "pcrs"); err != nil {
		return nil, err
	}

	pcrs := make(map[string][]int)
	for _, m := range data.Top {
		for k, l := range m {
			pcrs[k] = l
		}
	}

	return pcrs, nil
}

// CreatePCRPolicy creates a pcr policy from selection
func CreatePCRPolicy(ctx context.Context, selector PCRSelector) (File, error) {
	policyFile := NewFile(".policy")
	return policyFile, tpm(ctx, "createpolicy", nil, nil, "--policy-pcr", "-l", selector.String(), "-L", string(policyFile))
}

// CreatePrimary key
func CreatePrimary(ctx context.Context, hash HashAlgorithm, key KeyAlgorithm) (File, error) {
	//tpm2_createprimary -C e -g sha1 -G rsa -c primary.context
	file := NewFile(".primary.context")
	return file, tpm(ctx, "createprimary", nil, nil, "-C", "e", "-g", string(hash), "-G", string(key), "-c", string(file))
}

// Create creates an object
func Create(ctx context.Context, hash HashAlgorithm, data io.Reader, primary File, policy File) (Object, error) {
	//tpm2_create -g sha256 -u obj.pub -r obj.priv -C primary.context -L policy.digest -a "noda|adminwithpolicy|fixedparent|fixedtpm" -i secret.bin
	obj := Object{
		private: NewFile(".priv"),
		public:  NewFile(".pub"),
	}

	return obj, tpm(ctx,
		"create",
		data, nil,
		"-g", string(hash),
		"-u", string(obj.public),
		"-r", string(obj.private),
		"-C", string(primary),
		"-L", string(policy),
		"-a", "noda|adminwithpolicy|fixedparent|fixedtpm",
		"-i", "-",
	)
}

func Load(ctx context.Context, primary File, obj Object) (loaded File, err error) {
	// tpm2_load -C primary.context -u obj.pub -r obj.priv -c load.context
	file := NewFile(".load.context")
	return file, tpm(ctx,
		"load",
		nil, nil,
		"-C", string(primary),
		"-u", string(obj.public),
		"-r", string(obj.private),
		"-c", string(file),
	)
}

// EvictControl
func EvictControl(ctx context.Context, loaded *File, address Address) error {
	// tpm2_evictcontrol -C o -c load.context 0x81000000
	if loaded != nil {
		// set
		return tpm(ctx, "evictcontrol",
			nil, nil,
			"-C", "o",
			"-c", string(*loaded),
			fmt.Sprintf("0x%x", address),
		)
	}
	// evict
	return tpm(ctx, "evictcontrol",
		nil, nil,
		"-C", "o",
		fmt.Sprintf("0x%x", address),
	)
}

// Unseal object
func Unseal(ctx context.Context, address Address, pcrs PCRSelector) (File, error) {
	// tpm2_unseal -c 0x81000000 -p pcr:sha1:0,1,2 # -o secret.bin
	file := NewFile(".raw")
	return file, tpm(ctx, "unseal",
		nil, nil,
		"-c", fmt.Sprintf("0x%x", address),
		"-p", fmt.Sprintf("pcr:%s", pcrs),
		"-o", string(file),
	)
}
