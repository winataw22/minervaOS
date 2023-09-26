package capacity

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	pciDir = "/sys/bus/pci/devices"
)

// Subdevice is subdevice information to a PCI device
type Subdevice struct {
	// SubsystemVendorID is device subsystem vendor ID according to PCI database
	SubsystemVendorID uint16
	// SubsystemVendorID is device subsystem ID according to PCI database
	SubsystemDeviceID uint16
	// Name is subdevice name according to PCI database
	Name string
}

// Device is a PCI device
type Device struct {
	// ID is device id according to PCI database
	ID uint16
	// Name is device name according to PCI database
	Name string
	// Subdevices is a slice of all subdevices according to PCI database
	Subdevices []Subdevice
}

// Vendor is Vendor information
type Vendor struct {
	// ID of the vendor according to PCI database
	ID uint16
	// Name of the vendor
	Name string
	// All known devices by this vendor
	Devices map[uint16]Device
}

// make sure pci.ids is up to date
//go:generate curl -o pci/pci.ids -L https://pci-ids.ucw.cz/v2.2/pci.ids

var (
	//go:embed pci/pci.ids
	src []byte

	pattern = regexp.MustCompile(`^(\t*)([\d|a-f]{4}) ([\d|a-f]{4}){0,1}(.+)$`)
	data    map[uint16]Vendor

	gpuVendorsWhitelist = []uint16{
		0x1002, // AMD
		0x10de, // NVIDIA
	}
)

func init() {
	var err error
	data, err = loadVendors(src)
	if err != nil {
		panic(fmt.Sprintf("failed to build vendors db: %s", err))
	}
}

func loadVendors(src []byte) (map[uint16]Vendor, error) {
	vendors := make(map[uint16]Vendor)

	scanner := bufio.NewScanner(bytes.NewBuffer(src))
	var current *Vendor
	var deviceID uint16
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}

		matches := pattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		// this should have `5 parts`
		tabs := matches[1]
		id64, err := strconv.ParseUint(matches[2], 16, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vendor or device id: %w", err)
		}
		rest := strings.TrimSpace(matches[4])

		id := uint16(id64)
		// the line has no leading tabs. this is a new vendor
		if len(tabs) == 0 {
			if current != nil {
				vendors[current.ID] = *current
			}
			current = &Vendor{
				ID:      id,
				Name:    rest,
				Devices: make(map[uint16]Device),
			}
		} else if len(tabs) == 1 {
			// that is a device
			if current == nil {
				return nil, fmt.Errorf("device appeared before a vendor: %s", line)
			}

			current.Devices[id] = Device{
				ID:         id,
				Name:       rest,
				Subdevices: make([]Subdevice, 0),
			}
			deviceID = id
		} else if len(tabs) == 2 {
			if current == nil {
				return nil, fmt.Errorf("subsystem appeared before a vendor: %s", line)
			}
			if len(current.Devices) == 0 {
				return nil, fmt.Errorf("subsystem appeared before a device: %s", line)
			}
			ssid64, err := strconv.ParseUint(matches[3], 16, 16)
			if err != nil {
				return nil, fmt.Errorf("failed to parse subsystem id: %w", err)
			}
			ssid := uint16(ssid64)

			device := current.Devices[deviceID]
			device.Subdevices = append(device.Subdevices, Subdevice{
				SubsystemVendorID: id,
				SubsystemDeviceID: ssid,
				Name:              rest,
			})
			current.Devices[deviceID] = device

		}
	}

	return vendors, nil
}

// GetVendor gets a vendor object given Vendor ID
func GetVendor(vendor uint16) (Vendor, bool) {
	v, ok := data[vendor]
	return v, ok
}

// GetDevice looks up the devices db to given Vendor and Device IDs
func GetDevice(vendor uint16, device uint16) (v Vendor, d Device, ok bool) {
	v, ok = data[vendor]
	if !ok {
		return
	}
	d, ok = v.Devices[device]

	return
}

// GetSubdevice looks up the subdevice using devices db
func GetSubdevice(vendor uint16, device uint16, subsystemVendorID uint16, subsystemDeviceID uint16) (Subdevice, bool) {
	_, d, ok := GetDevice(vendor, device)
	if !ok {
		return Subdevice{}, false
	}
	for _, subdevice := range d.Subdevices {
		if subdevice.SubsystemDeviceID == subsystemDeviceID && subdevice.SubsystemVendorID == subsystemVendorID {
			return subdevice, true
		}
	}
	return Subdevice{}, false
}

// Filter over the PCI for listing
type Filter func(pci *PCI) bool

// PCI device
type PCI struct {
	// Slot is the PCI device slot
	Slot string
	// Vendor of the device
	Vendor uint16
	// Device id
	Device uint16
	// Class of the device
	Class uint32
	// Subsystem Vendor of the device
	SubsystemVendor uint16
	// Subsystem ID of the device
	SubsystemDevice uint16
}

// GetDevice gets the attached PCI device information (vendor and device)
func (p *PCI) GetDevice() (Vendor, Device, bool) {
	return GetDevice(p.Vendor, p.Device)
}

// GetSubdevice gets the attached PCI subdevice information
func (p *PCI) GetSubdevice() (Subdevice, bool) {
	return GetSubdevice(p.Vendor, p.Device, p.SubsystemVendor, p.SubsystemDevice)
}

// ShortID returns a short identification string
// for the device in the format `slot/vendor/device`
func (p *PCI) ShortID() string {
	return fmt.Sprintf("%s/%04x/%04x", p.Slot, p.Vendor, p.Device)
}

func (p PCI) String() string {
	return fmt.Sprintf("%s %04x: [%04x:%04x]", p.Slot, p.Class, p.Vendor, p.Device)
}

// Flag read a custom flag on PCI device as uint64
func (p *PCI) Flag(name string) (uint64, error) {
	return readUint64(filepath.Join(pciDir, p.Slot, name), 64)
}

func pciDeviceFromSlot(slot string) (PCI, error) {
	const (
		classFile           = "class"
		vendorFile          = "vendor"
		deviceFile          = "device"
		subsystemVendorFile = "subsystem_vendor"
		subsystemDeviceFile = "subsystem_device"
	)
	class, err := readUint64(filepath.Join(pciDir, slot, classFile), 32)
	if err != nil {
		return PCI{}, fmt.Errorf("failed to get device '%s' class: %w", slot, err)
	}
	vendor, err := readUint64(filepath.Join(pciDir, slot, vendorFile), 16)
	if err != nil {
		return PCI{}, fmt.Errorf("failed to get device '%s' vendor: %w", slot, err)
	}

	device, err := readUint64(filepath.Join(pciDir, slot, deviceFile), 16)
	if err != nil {
		return PCI{}, fmt.Errorf("failed to get device '%s' device: %w", slot, err)
	}
	subsystemVendor, err := readUint64(filepath.Join(pciDir, slot, subsystemVendorFile), 16)
	if err != nil {
		return PCI{}, fmt.Errorf("failed to get device '%s' subsystem vendor: %w", slot, err)
	}
	subsystemDevice, err := readUint64(filepath.Join(pciDir, slot, subsystemDeviceFile), 16)
	if err != nil {
		return PCI{}, fmt.Errorf("failed to get device '%s' subsystem device: %w", slot, err)
	}

	pci := PCI{
		Slot:            slot,
		Class:           uint32(class),
		Vendor:          uint16(vendor),
		Device:          uint16(device),
		SubsystemVendor: uint16(subsystemVendor),
		SubsystemDevice: uint16(subsystemDevice),
	}

	return pci, err
}

// ListPCI lists all PCI devices attached to the machine, applying provided filters
func ListPCI(filter ...Filter) ([]PCI, error) {
	var devices []PCI
	entries, err := os.ReadDir(pciDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sys pci devices: %w", err)
	}

next:
	for _, entry := range entries {
		pci, err := pciDeviceFromSlot(entry.Name())
		if err != nil {
			return nil, err
		}

		for _, f := range filter {
			if !f(&pci) {
				continue next
			}
		}

		devices = append(devices, pci)
	}

	return devices, nil
}

func readUint64(path string, bitSize int) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("failed to read path '%s': %w", path, err)
	}

	return strconv.ParseUint(
		strings.TrimPrefix(
			strings.TrimSpace(string(data)),
			"0x"),
		16,
		bitSize)
}

// GPU Filter only devices with GPU capabilities
// NOTE: we only also now white list NVIDIA and AMD
// we skip integrated GPU  normally on slots `0000:00:XX.X` and allow only discrete ones normally on slot `0000:XX:YY.ZZ`
func GPU(p *PCI) bool {
	return p.Class == 0x030000 && in(p.Vendor, gpuVendorsWhitelist) && !strings.HasPrefix(p.Slot, "0000:00:")
}

// PCIBridge returns true if p is a PCI bridge
func PCIBridge(p *PCI) bool {
	// this will include 0x060000 and 0x060400
	// or any other pci bridge device
	return p.Class>>16 == 0x06
}

// Not negates a filter
func Not(f Filter) Filter {
	return func(pci *PCI) bool {
		return !f(pci)
	}
}

// IoMMUGroup given a pci device, return all devices in the same iommu group
func IoMMUGroup(pci PCI, filter ...Filter) ([]PCI, error) {
	path := filepath.Join(pciDir, pci.Slot, "iommu_group", "devices")
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		// no groups
		return []PCI{pci}, nil
	} else if err != nil {
		return nil, err
	}

	var devices []PCI
next:
	for _, entry := range entries {
		pci, err := pciDeviceFromSlot(entry.Name())
		if err != nil {
			return nil, err
		}
		for _, f := range filter {
			if !f(&pci) {
				continue next
			}
		}

		devices = append(devices, pci)
	}

	return devices, nil
}

func in[T comparable](v T, l []T) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}

	return false
}
