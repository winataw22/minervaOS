package cloudinit

type marsh map[string]interface{}

type NetworkObjectType string
type SubnetType string
type MountType string

const (
	MountTypeAuto     = "auto"
	MountTypeVirtiofs = "virtiofs"
)

type Metadata struct {
	InstanceID string `yaml:"instance-id"`
	Hostname   string `yaml:"local-hostname"`
}

type User struct {
	Name string   `yaml:"name"`
	Keys []string `yaml:"ssh_authorized_keys"`
}

type Mount struct {
	Source string
	Target string
	Type   MountType
}

func (m Mount) MarshalYAML() (interface{}, error) {
	t := m.Type
	if len(t) == 0 {
		t = MountTypeAuto
	}
	return []string{
		m.Source,
		m.Target,
		string(t),
		"defaults",
		"0",
		"0",
	}, nil
}

type Extension struct {
	Entrypoint  string
	Environment map[string]string
}

type Route struct {
	To     string `yaml:"to"`
	Via    string `yaml:"via"`
	Metric int    `yaml:"metric,omitempty"`
}

type MacMatch string

func (n MacMatch) String() string {
	return string(n)
}

func (n MacMatch) MarshalYAML() (interface{}, error) {
	return marsh{
		"macaddress": string(n),
	}, nil
}

type Nameservers struct {
	Search    []string `yaml:"search,omitempty"`
	Addresses []string `yaml:"addresses"` //required
}

type Ethernet struct {
	Name        string       `yaml:"-"`
	Mac         MacMatch     `yaml:"match"`
	DHCP4       bool         `yaml:"dhcp4"`
	Addresses   []string     `yaml:"addresses"`
	Gateway4    string       `yaml:"gateway4,omitempty"`
	Gateway6    string       `yaml:"gateway6,omitempty"`
	Routes      []Route      `yaml:"routes,omitempty"`
	Nameservers *Nameservers `yaml:"nameservers,omitempty"`
}

type Configuration struct {
	Metadata  Metadata
	Network   []Ethernet
	Users     []User
	Mounts    []Mount
	Extension Extension
}
