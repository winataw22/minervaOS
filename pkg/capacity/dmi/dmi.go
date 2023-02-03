package dmi

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// DecoderVersion is the information about the decoder in this package
const DecoderVersion = `0-core Go dmi decoder v0.2.0`

// Type (allowed types 0 -> 42)
type Type int

// DMI represents a map of SectionTypeStr to Section parsed from dmidecode output,
// as well as information about the tool used to get these sections
// Property in section is in the form of key value pairs where values are optional
// and may include a list of items as well.
// k: [v]
//
//	[
//		item1
//		item2
//		...
//	]
type DMI struct {
	Tooling  Tooling   `json:"tooling"`
	Sections []Section `json:"sections"`
}

// Tooling holds the information and version about the tool used to
// read DMI information
type Tooling struct {
	Aggregator string `json:"aggregator"`
	Decoder    string `json:"decoder"`
}

// List of DMI section
const (
	TypeBIOS Type = iota
	TypeSystem
	TypeBaseboard
	TypeChassis
	TypeProcessor
	TypeMemoryController
	TypeMemoryModule
	TypeCache
	TypePortConnector
	TypeSystemSlots
	TypeOnBoardDevices
	TypeOEMSettings
	TypeSystemConfigurationOptions
	TypeBIOSLanguage
	TypeGroupAssociations
	TypeSystemEventLog
	TypePhysicalMemoryArray
	TypeMemoryDevice
	Type32BitMemoryError
	TypeMemoryArrayMappedAddress
	TypeMemoryDeviceMappedAddress
	TypeBuiltinPointingDevice
	TypePortableBattery
	TypeSystemReset
	TypeHardwareSecurity
	TypeSystemPowerControls
	TypeVoltageProbe
	TypeCoolingDevice
	TypeTemperatureProbe
	TypeElectricalCurrentProbe
	TypeOutOfBandRemoteAccess
	TypeBootIntegrityServices
	TypeSystemBoot
	Type64BitMemoryError
	TypeManagementDevice
	TypeManagementDeviceComponent
	TypeManagementDeviceThresholdData
	TypeMemoryChannel
	TypeIPMIDevice
	TypePowerSupply
	TypeAdditionalInformation
	TypeOnboardDevicesExtendedInformation
	TypeManagementControllerHostInterface
)

var dmitypeToString = map[Type]string{
	TypeBIOS:                              "BIOS",
	TypeSystem:                            "System",
	TypeBaseboard:                         "Baseboard",
	TypeChassis:                           "Chassis",
	TypeProcessor:                         "Processor",
	TypeMemoryController:                  "MemoryController",
	TypeMemoryModule:                      "MemoryModule",
	TypeCache:                             "Cache",
	TypePortConnector:                     "PortConnector",
	TypeSystemSlots:                       "SystemSlots",
	TypeOnBoardDevices:                    "OnBoardDevices",
	TypeOEMSettings:                       "OEMSettings",
	TypeSystemConfigurationOptions:        "SystemConfigurationOptions",
	TypeBIOSLanguage:                      "BIOSLanguage",
	TypeGroupAssociations:                 "GroupAssociations",
	TypeSystemEventLog:                    "SystemEventLog",
	TypePhysicalMemoryArray:               "PhysicalMemoryArray",
	TypeMemoryDevice:                      "MemoryDevice",
	Type32BitMemoryError:                  "32BitMemoryError",
	TypeMemoryArrayMappedAddress:          "MemoryArrayMappedAddress",
	TypeMemoryDeviceMappedAddress:         "MemoryDeviceMappedAddress",
	TypeBuiltinPointingDevice:             "BuiltinPointingDevice",
	TypePortableBattery:                   "PortableBattery",
	TypeSystemReset:                       "SystemReset",
	TypeHardwareSecurity:                  "HardwareSecurity",
	TypeSystemPowerControls:               "SystemPowerControls",
	TypeVoltageProbe:                      "VoltageProbe",
	TypeCoolingDevice:                     "CoolingDevice",
	TypeTemperatureProbe:                  "TempratureProbe",
	TypeElectricalCurrentProbe:            "ElectricalCurrentProbe",
	TypeOutOfBandRemoteAccess:             "OutOfBandRemoteAccess",
	TypeBootIntegrityServices:             "BootIntegrityServices",
	TypeSystemBoot:                        "SystemBoot",
	Type64BitMemoryError:                  "64BitMemoryError",
	TypeManagementDevice:                  "ManagementDevice",
	TypeManagementDeviceComponent:         "ManagementDeviceComponent",
	TypeManagementDeviceThresholdData:     "ManagementThresholdData",
	TypeMemoryChannel:                     "MemoryChannel",
	TypeIPMIDevice:                        "IPMIDevice",
	TypePowerSupply:                       "PowerSupply",
	TypeAdditionalInformation:             "AdditionalInformation",
	TypeOnboardDevicesExtendedInformation: "OnboardDeviceExtendedInformation",
	TypeManagementControllerHostInterface: "ManagementControllerHostInterface",
}

var dmiTypeRegex = regexp.MustCompile("DMI type ([0-9]+)")
var kvRegex = regexp.MustCompile("(.+?):(.*)")

// Decode run and parse the dmidecode command
func Decode() (*DMI, error) {
	output, err := exec.Command("dmidecode").Output()
	if err != nil {
		return nil, err
	}
	return parseDMI(string(output))

}

func (d *DMI) BoardVersion() string {
	if len(d.Sections) < int(TypeBaseboard) {
		return ""
	}

	board := d.Sections[TypeBaseboard]
	if len(board.SubSections) < 1 {
		return ""
	}

	return board.SubSections[0].Properties["Serial Number"].Val
}

// dmiTypeToString returns string representation of Type t
func dmiTypeToString(t Type) string {
	str := dmitypeToString[t]
	if str == "" {
		return fmt.Sprintf("Custom Type %d", t)
	}
	return str
}

// Extract the DMI type from the handleline.
func getTypeFromHandleLine(line string) (Type, error) {
	m := dmiTypeRegex.FindStringSubmatch(line)
	if len(m) == 2 {
		t, err := strconv.Atoi(m[1])
		return Type(t), err
	}
	return 0, fmt.Errorf("couldn't find dmitype in handleline %s", line)
}

func getLineLevel(line string) int {
	for i, c := range line {
		if !unicode.IsSpace(c) {
			return i
		}
	}
	return 0
}

func propertyFromLine(line string) (string, PropertyData, error) {
	m := kvRegex.FindStringSubmatch(line)
	if len(m) == 3 {
		k, v := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		return k, PropertyData{Val: v}, nil
	} else if len(m) == 2 {
		k := strings.TrimSpace(m[1])
		return k, PropertyData{Val: ""}, nil
	} else {
		return "", PropertyData{}, fmt.Errorf("couldn't find key value pair on the line %s", line)
	}
}

// PropertyData represents a key value pair with optional list of items
type PropertyData struct {
	Val   string   `json:"value"`
	Items []string `json:"items,omitempty"`
}

// Section represents a complete section like BIOS or Baseboard
type Section struct {
	HandleLine  string       `json:"handleline"`
	TypeStr     string       `json:"typestr,omitempty"`
	Type        Type         `json:"typenum"`
	SubSections []SubSection `json:"subsections"`
}

// SubSection represents part of a section, identified by a title
type SubSection struct {
	Title      string                  `json:"title"`
	Properties map[string]PropertyData `json:"properties,omitempty"`
}

func newSection() Section {
	return Section{}
}

func readSection(section *Section, lines []string, start int) (int, error) {
	if (start + 2) > len(lines) {
		return 0, fmt.Errorf("invalid section size")
	}

	section.HandleLine = lines[start]
	start++
	dmitype, err := getTypeFromHandleLine(section.HandleLine)

	if err != nil {
		return 0, err
	}

	section.Type = dmitype
	section.TypeStr = dmiTypeToString(dmitype)

	subSection := SubSection{
		Title:      lines[start],
		Properties: make(map[string]PropertyData),
	}
	start++

	for start < len(lines) {
		line := lines[start]
		if strings.TrimSpace(line) == "" {
			section.SubSections = append(section.SubSections, subSection)
			return start, nil
		}
		if !unicode.IsSpace([]rune(line)[0]) {
			section.SubSections = append(section.SubSections, subSection)
			subSection = SubSection{
				Title:      line,
				Properties: make(map[string]PropertyData),
			}
			start++
			continue
		}
		indentLevel := getLineLevel(line)
		key, propertyData, err := propertyFromLine(line)
		if err != nil {
			return 0, err
		}
		nxtIndentLevel := 0
		if len(lines) > start+1 {
			nxtIndentLevel = getLineLevel(lines[start+1])
		}

		if nxtIndentLevel > indentLevel {
			start = readList(&propertyData, lines, start+1)
		}

		start++
		subSection.Properties[key] = propertyData
	}
	section.SubSections = append(section.SubSections, subSection)
	return start, nil
}

func readList(propertyData *PropertyData, lines []string, start int) int {
	startIndentLevel := getLineLevel(lines[start])
	for start < len(lines) {
		line := lines[start]
		indentLevel := getLineLevel(line)

		if indentLevel >= startIndentLevel {
			propertyData.Items = append(propertyData.Items, strings.TrimSpace(line))
		} else {
			return start - 1
		}
		start++
	}
	return start
}

// parseDMI Parses dmidecode output into DMI structure
func parseDMI(input string) (*DMI, error) {
	lines := strings.Split(input, "\n")
	secs := []Section{}

	var (
		start   int
		tooling Tooling
	)

	for ; start < len(lines); start++ {
		if strings.HasPrefix(lines[start], "Handle") {
			// do not skip line, we want to start at this one,
			// in our next loop
			break
		}
		if strings.HasPrefix(lines[start], "#") {
			tooling.Aggregator = strings.TrimSpace(strings.TrimPrefix(lines[start], "#"))
			start++ // skip line, we already consumed it
			break
		}
	}
	if tooling.Aggregator == "" {
		tooling.Aggregator = "unknown"
	}
	tooling.Decoder = DecoderVersion // include decoder tool information

	for ; start < len(lines); start++ {
		line := lines[start]
		if strings.HasPrefix(line, "Handle") {
			section := newSection()
			var err error
			start, err = readSection(&section, lines, start)
			if err != nil {
				return nil, err
			}
			secs = append(secs, section)
		}
	}

	return &DMI{
		Tooling:  tooling,
		Sections: secs,
	}, nil
}
