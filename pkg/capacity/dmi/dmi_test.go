package dmi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	sample1 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0001, DMI type 1, 27 bytes
System Information
		Manufacturer: LENOVO
		Product Name: 20042
		Version: Lenovo G560
		Serial Number: 2677240001087
		UUID: CB3E6A50-A77B-E011-88E9-B870F4165734
		Wake-up Type: Power Switch
		SKU Number: Calpella_CRB
		Family: Intel_Mobile
	`
	sample2 = `
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0000, DMI type 0, 24 bytes
BIOS Information
		Vendor: LENOVO
		Version: 29CN40WW(V2.17)
		Release Date: 04/13/2011
		ROM Size: 2048 kB
		Characteristics:
				PCI is supported
				BIOS is upgradeable
				BIOS shadowing is allowed
				Boot from CD is supported
				Selectable boot is supported
				EDD is supported
				Japanese floppy for NEC 9800 1.2 MB is supported (int 13h)
				Japanese floppy for Toshiba 1.2 MB is supported (int 13h)
				5.25"/360 kB floppy services are supported (int 13h)
				5.25"/1.2 MB floppy services are supported (int 13h)
				3.5"/720 kB floppy services are supported (int 13h)
				3.5"/2.88 MB floppy services are supported (int 13h)
				8042 keyboard services are supported (int 9h)
				CGA/mono video services are supported (int 10h)
				ACPI is supported
				USB legacy is supported
				BIOS boot specification is supported
				Targeted content distribution is supported
		BIOS Revision: 1.40
	`
	sample3 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0001, DMI type 1, 27 bytes
System Information
		Manufacturer: LENOVO
		Product Name: 20042
		Version: Lenovo G560
		Serial Number: 2677240001087
		UUID: CB3E6A50-A77B-E011-88E9-B870F4165734
		Wake-up Type: Power Switch
		SKU Number: Calpella_CRB
		Family: Intel_Mobile

Handle 0x000D, DMI type 12, 5 bytes
System Configuration Options
		Option 1: String1 for Type12 Equipment Manufacturer
		Option 2: String2 for Type12 Equipment Manufacturer
		Option 3: String3 for Type12 Equipment Manufacturer
		Option 4: String4 for Type12 Equipment Manufacturer

Handle 0x000E, DMI type 15, 29 bytes
System Event Log
		Area Length: 0 bytes
		Header Start Offset: 0x0000
		Data Start Offset: 0x0000
		Access Method: General-purpose non-volatile data functions
		Access Address: 0x0000
		Status: Valid, Not Full
		Change Token: 0x12345678
		Header Format: OEM-specific
		Supported Log Type Descriptors: 3
		Descriptor 1: POST memory resize
		Data Format 1: None
		Descriptor 2: POST error
		Data Format 2: POST results bitmap
		Descriptor 3: Log area reset/cleared
		Data Format 3: None

Handle 0x0011, DMI type 32, 20 bytes
System Boot Information
		Status: No errors detected
	`
	sample4 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 2.6 present.

Handle 0x0000, DMI type 0, 24 bytes
BIOS Information
		Vendor: LENOVO
		Version: 29CN40WW(V2.17)
		Release Date: 04/13/2011
		ROM Size: 2048 kB
		Characteristics:
				PCI is supported
				BIOS is upgradeable
				BIOS shadowing is allowed
				Boot from CD is supported
				Selectable boot is supported
				EDD is supported
				Japanese floppy for NEC 9800 1.2 MB is supported (int 13h)
				Japanese floppy for Toshiba 1.2 MB is supported (int 13h)
				5.25"/360 kB floppy services are supported (int 13h)
				5.25"/1.2 MB floppy services are supported (int 13h)
				3.5"/720 kB floppy services are supported (int 13h)
				3.5"/2.88 MB floppy services are supported (int 13h)
				8042 keyboard services are supported (int 9h)
				CGA/mono video services are supported (int 10h)
				ACPI is supported
				USB legacy is supported
				BIOS boot specification is supported
				Targeted content distribution is supported
		BIOS Revision: 1.40

Handle 0x002C, DMI type 4, 42 bytes
Processor Information
		Socket Designation: CPU
		Type: Central Processor
		Family: Core 2 Duo
		Manufacturer: Intel(R) Corporation
		ID: 55 06 02 00 FF FB EB BF
		Signature: Type 0, Family 6, Model 37, Stepping 5
		Flags:
				FPU (Floating-point unit on-chip)
				VME (Virtual mode extension)
				DE (Debugging extension)
				PSE (Page size extension)
				TSC (Time stamp counter)
				MSR (Model specific registers)
				PAE (Physical address extension)
				MCE (Machine check exception)
				CX8 (CMPXCHG8 instruction supported)
				APIC (On-chip APIC hardware supported)
				SEP (Fast system call)
				MTRR (Memory type range registers)
				PGE (Page global enable)
				MCA (Machine check architecture)
				CMOV (Conditional move instruction supported)
				PAT (Page attribute table)
				PSE-36 (36-bit page size extension)
				CLFSH (CLFLUSH instruction supported)
				DS (Debug store)
				ACPI (ACPI supported)
				MMX (MMX technology supported)
				FXSR (FXSAVE and FXSTOR instructions supported)
				SSE (Streaming SIMD extensions)
				SSE2 (Streaming SIMD extensions 2)
				SS (Self-snoop)
				HTT (Multi-threading)
				TM (Thermal monitor supported)
				PBE (Pending break enabled)
		Version: Intel(R) Core(TM) i3 CPU       M 370  @ 2.40GHz
		Voltage: 0.0 V
		External Clock: 1066 MHz
		Max Speed: 2400 MHz
		Current Speed: 2399 MHz
		Status: Populated, Enabled
		Upgrade: ZIF Socket
		L1 Cache Handle: 0x0030
		L2 Cache Handle: 0x002F
		L3 Cache Handle: 0x002D
		Serial Number: Not Specified
		Asset Tag: FFFF
		Part Number: Not Specified
		Core Count: 2
		Core Enabled: 2
		Thread Count: 4
		Characteristics:
				64-bit capable
	`

	sample5 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 3.0.0 present.
Table at 0x7AEAA000.

Handle 0x0014, DMI type 10, 20 bytes
On Board Device 1 Information
	Type: Video
	Status: Enabled
	Description:  Intel(R) HD Graphics Device
On Board Device 2 Information
	Type: Ethernet
	Status: Enabled
	Description:  Intel(R) I219-V Gigabit Network Device
On Board Device 3 Information
	Type: Sound
	Status: Enabled
	Description:  Realtek High Definition Audio Device
On Board Device 4 Information
	Type: Other
	Status: Enabled
	Description: CIR Device
On Board Device 5 Information
	Type: Other
	Status: Enabled
	Description: SD
On Board Device 6 Information
	Type: Other
	Status: Enabled
	Description: Intel Dual Band Wireless-AC 8265
On Board Device 7 Information
	Type: Other
	Status: Enabled
	Description: Bluetooth
On Board Device 8 Information
	Type: Other
	Status: Disabled
	Description: Thunderbolt
`

	sample6 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 3.1 present.
188 structures occupying 9476 bytes.
Table at 0x74921000.

Handle 0x0076, DMI type 216, 23 bytes
OEM-specific Type
	Header and Data:
		D8 17 76 00 06 00 01 02 05 20 31 2E 33 2E 31 39
		00 00 00 00 00 00 00
	Strings:
		Chassis Firmware
			1.3.19
`

	sample7 = `
# dmidecode 3.1
Getting SMBIOS data from sysfs.
SMBIOS 3.0.0 present.
Table at 0x7AEAA000.

Handle 0x0036, DMI type 17, 40 bytes
Memory Device
	Array Handle: 0x0035
	Error Information Handle: Not Provided
	Total Width: 64 bits
	Data Width: 64 bits
	Size: 8192 MB
	Form Factor: SODIMM
	Set: None
	Locator: ChannelA-DIMM0
	Bank Locator: BANK 0
	Type: DDR4
	Type Detail: Synchronous Unbuffered (Unregistered)
	Speed: 2400 MT/s
	Manufacturer: 859B
	Serial Number: E0AD159C
	Asset Tag: 9876543210
	Part Number: CT8G4SFD824A.M16FB  
	Rank: 2
	Configured Clock Speed: 2133 MT/s
	Minimum Voltage: 1.2 V
	Maximum Voltage: 1.2 V
	Configured Voltage: 1.2 V

Handle 0x0037, DMI type 17, 40 bytes
Memory Device
	Array Handle: 0x0035
	Error Information Handle: Not Provided
	Total Width: 64 bits
	Data Width: 64 bits
	Size: 8192 MB
	Form Factor: SODIMM
	Set: None
	Locator: ChannelB-DIMM0
	Bank Locator: BANK 2
	Type: DDR4
	Type Detail: Synchronous Unbuffered (Unregistered)
	Speed: 2400 MT/s
	Manufacturer: 859B
	Serial Number: E0AD159D
	Asset Tag: 9876543210
	Part Number: CT8G4SFD824A.M16FB  
	Rank: 2
	Configured Clock Speed: 2133 MT/s
	Minimum Voltage: 1.2 V
	Maximum Voltage: 1.2 V
	Configured Voltage: 1.2 V
`
)

var biosInfoTests = map[string]string{
	"Vendor":          "LENOVO",
	"Version":         "29CN40WW(V2.17)",
	"Release Date":    "04/13/2011",
	"ROM Size":        "2048 kB",
	"Characteristics": "",
	"BIOS Revision":   "1.40",
}
var sysInfoTests = map[string]string{
	"Manufacturer":  "LENOVO",
	"Product Name":  "20042",
	"Version":       "Lenovo G560",
	"Serial Number": "2677240001087",
	"UUID":          "CB3E6A50-A77B-E011-88E9-B870F4165734",
	"Wake-up Type":  "Power Switch",
	"SKU Number":    "Calpella_CRB",
	"Family":        "Intel_Mobile",
}

var sysConfigurationTests = map[string]string{
	"Option 1": "String1 for Type12 Equipment Manufacturer",
	"Option 2": "String2 for Type12 Equipment Manufacturer",
	"Option 3": "String3 for Type12 Equipment Manufacturer",
	"Option 4": "String4 for Type12 Equipment Manufacturer",
}

var sysEventLogTests = map[string]string{
	"Area Length":                    "0 bytes",
	"Header Start Offset":            "0x0000",
	"Data Start Offset":              "0x0000",
	"Access Method":                  "General-purpose non-volatile data functions",
	"Access Address":                 "0x0000",
	"Status":                         "Valid, Not Full",
	"Change Token":                   "0x12345678",
	"Header Format":                  "OEM-specific",
	"Supported Log Type Descriptors": "3",
	"Descriptor 1":                   "POST memory resize",
	"Data Format 1":                  "None",
	"Descriptor 2":                   "POST error",
	"Data Format 2":                  "POST results bitmap",
	"Descriptor 3":                   "Log area reset/cleared",
	"Data Format 3":                  "None",
}

var sysBootTests = map[string]string{
	"Status": "No errors detected",
}

var processorTests = map[string]string{
	"Socket Designation": "CPU",
	"Type":               "Central Processor",
	"Family":             "Core 2 Duo",
	"Manufacturer":       "Intel(R) Corporation",
	"ID":                 "55 06 02 00 FF FB EB BF",
	"Signature":          "Type 0, Family 6, Model 37, Stepping 5",
	"Flags":              "",
	"Version":            "Intel(R) Core(TM) i3 CPU       M 370  @ 2.40GHz",
	"Voltage":            "0.0 V",
	"External Clock":     "1066 MHz",
	"Max Speed":          "2400 MHz",
	"Current Speed":      "2399 MHz",
	"Status":             "Populated, Enabled",
	"Upgrade":            "ZIF Socket",
	"L1 Cache Handle":    "0x0030",
	"L2 Cache Handle":    "0x002F",
	"L3 Cache Handle":    "0x002D",
	"Serial Number":      "Not Specified",
	"Asset Tag":          "FFFF",
	"Part Number":        "Not Specified",
	"Core Count":         "2",
	"Core Enabled":       "2",
	"Thread Count":       "4",
	"Characteristics":    "",
}

var onBoardDevicesTests = []SubSection{
	{
		Title: "On Board Device 1 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "Intel(R) HD Graphics Device",
			},
			"Status": {
				Val: "Enabled",
			},
			"Type": {
				Val: "Video",
			},
		},
	},
	{
		Title: "On Board Device 2 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "Intel(R) I219-V Gigabit Network Device",
			},
			"Status": {
				Val: "Enabled",
			},
			"Type": {
				Val: "Ethernet",
			},
		},
	}, {
		Title: "On Board Device 3 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "Realtek High Definition Audio Device",
			},
			"Status": {
				Val: "Enabled",
			},
			"Type": {
				Val: "Sound",
			},
		},
	}, {
		Title: "On Board Device 4 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "CIR Device",
			},
			"Status": {
				Val: "Enabled",
			},
			"Type": {
				Val: "Other",
			},
		},
	}, {
		Title: "On Board Device 5 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "SD",
			},
			"Status": {
				Val: "Enabled",
			},
			"Type": {
				Val: "Other",
			},
		},
	}, {
		Title: "On Board Device 6 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "Intel Dual Band Wireless-AC 8265",
			},
			"Status": {
				Val: "Enabled",
			},
			"Type": {
				Val: "Other",
			},
		},
	}, {
		Title: "On Board Device 7 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "Bluetooth",
			},
			"Status": {
				Val: "Enabled",
			},
			"Type": {
				Val: "Other",
			},
		},
	}, {
		Title: "On Board Device 8 Information",
		Properties: map[string]PropertyData{
			"Description": {
				Val: "Thunderbolt",
			},
			"Status": {
				Val: "Disabled",
			},
			"Type": {
				Val: "Other",
			},
		},
	},
}

var oemSpecificTypeWithIndentedListItems = SubSection{
	Title: "OEM-specific Type",
	Properties: map[string]PropertyData{
		"Header and Data": {
			Val: "",
			Items: []string{
				"D8 17 76 00 06 00 01 02 05 20 31 2E 33 2E 31 39",
				"00 00 00 00 00 00 00",
			},
		},
		"Strings": {
			Val: "",
			Items: []string{
				"Chassis Firmware",
				"1.3.19",
			},
		},
	},
}

var multpleSectionsWithSameTypeStr = []SubSection{
	{
		Title: "Memory Device",
		Properties: map[string]PropertyData{
			"Array Handle":             {Val: "0x0035"},
			"Error Information Handle": {Val: "Not Provided"},
			"Total Width":              {Val: "64 bits"},
			"Data Width":               {Val: "64 bits"},
			"Size":                     {Val: "8192 MB"},
			"Form Factor":              {Val: "SODIMM"},
			"Set":                      {Val: "None"},
			"Locator":                  {Val: "ChannelA-DIMM0"},
			"Bank Locator":             {Val: "BANK 0"},
			"Type":                     {Val: "DDR4"},
			"Type Detail":              {Val: "Synchronous Unbuffered (Unregistered)"},
			"Speed":                    {Val: "2400 MT/s"},
			"Manufacturer":             {Val: "859B"},
			"Serial Number":            {Val: "E0AD159C"},
			"Asset Tag":                {Val: "9876543210"},
			"Part Number":              {Val: "CT8G4SFD824A.M16FB"},
			"Rank":                     {Val: "2"},
			"Configured Clock Speed":   {Val: "2133 MT/s"},
			"Minimum Voltage":          {Val: "1.2 V"},
			"Maximum Voltage":          {Val: "1.2 V"},
			"Configured Voltage":       {Val: "1.2 V"},
		},
	},
	{
		Title: "Memory Device",
		Properties: map[string]PropertyData{
			"Array Handle":             {Val: "0x0035"},
			"Error Information Handle": {Val: "Not Provided"},
			"Total Width":              {Val: "64 bits"},
			"Data Width":               {Val: "64 bits"},
			"Size":                     {Val: "8192 MB"},
			"Form Factor":              {Val: "SODIMM"},
			"Set":                      {Val: "None"},
			"Locator":                  {Val: "ChannelB-DIMM0"},
			"Bank Locator":             {Val: "BANK 2"},
			"Type":                     {Val: "DDR4"},
			"Type Detail":              {Val: "Synchronous Unbuffered (Unregistered)"},
			"Speed":                    {Val: "2400 MT/s"},
			"Manufacturer":             {Val: "859B"},
			"Serial Number":            {Val: "E0AD159D"},
			"Asset Tag":                {Val: "9876543210"},
			"Part Number":              {Val: "CT8G4SFD824A.M16FB"},
			"Rank":                     {Val: "2"},
			"Configured Clock Speed":   {Val: "2133 MT/s"},
			"Minimum Voltage":          {Val: "1.2 V"},
			"Maximum Voltage":          {Val: "1.2 V"},
			"Configured Voltage":       {Val: "1.2 V"},
		},
	},
}

// util functions to make it easier to fetch (sub) section(s)
func sectionsFromDMI(dmi *DMI, sectionTypeStr string) (sections []Section) {
	for _, section := range dmi.Sections {
		if section.TypeStr == sectionTypeStr {
			sections = append(sections, section)
		}
	}
	return
}
func subSectionsFromDMI(dmi *DMI, sectionTypeStr, subSectionTitle string) (subSections []SubSection) {
	for _, section := range dmi.Sections {
		if section.TypeStr == sectionTypeStr {
			for _, subSection := range section.SubSections {
				if subSection.Title == subSectionTitle {
					subSections = append(subSections, subSection)
				}
			}
		}
	}
	return
}

func TestParseSectionSimple(t *testing.T) {
	dmi, err := parseDMI(sample1)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, dmi.Sections, 1); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, subSectionsFromDMI(dmi, "System", "System Information")[0].Properties, 8); !ok {
		t.Fatal()
	}
	for k, v := range sysInfoTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "System", "System Information")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}

}
func TestParseSectionWithListProperty(t *testing.T) {
	dmi, err := parseDMI(sample2)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "unknown"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, dmi.Sections, 1); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, subSectionsFromDMI(dmi, "BIOS", "BIOS Information")[0].Properties, 6); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, subSectionsFromDMI(dmi, "BIOS", "BIOS Information")[0].Properties["Characteristics"].Items, 18); !ok {
		t.Fatal()
	}

	for k, v := range biosInfoTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "BIOS", "BIOS Information")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}

}

func TestParseMultipleSectionsSimple(t *testing.T) {
	dmi, err := parseDMI(sample3)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, dmi.Sections, 4); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, subSectionsFromDMI(dmi, "System", "System Information")[0].Properties, 8); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, subSectionsFromDMI(dmi, "SystemEventLog", "System Event Log")[0].Properties, 15); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, TypeSystemBoot, sectionsFromDMI(dmi, "SystemBoot")[0].Type); !ok {
		t.Fatal()
	}

	for k, v := range sysInfoTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "System", "System Information")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}
	for k, v := range sysConfigurationTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "SystemConfigurationOptions", "System Configuration Options")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}
	for k, v := range sysEventLogTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "SystemEventLog", "System Event Log")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}
	for k, v := range sysBootTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "SystemBoot", "System Boot Information")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}

}
func TestParseMultipleSectionsWithListProperties(t *testing.T) {
	dmi, err := parseDMI(sample4)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, dmi.Sections, 2); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, subSectionsFromDMI(dmi, "BIOS", "BIOS Information")[0].Properties, 6); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, subSectionsFromDMI(dmi, "BIOS", "BIOS Information")[0].Properties["Characteristics"].Items, 18); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, subSectionsFromDMI(dmi, "Processor", "Processor Information")[0].Properties, 24); !ok {
		t.Fatal()
	}

	if ok := assert.Len(t, subSectionsFromDMI(dmi, "Processor", "Processor Information")[0].Properties["Flags"].Items, 28); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, "FPU (Floating-point unit on-chip)", subSectionsFromDMI(dmi, "Processor", "Processor Information")[0].Properties["Flags"].Items[0]); !ok {
		t.Fatal()
	}

	for k, v := range biosInfoTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "BIOS", "BIOS Information")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}

	for k, v := range processorTests {
		if ok := assert.Equal(t, v, subSectionsFromDMI(dmi, "Processor", "Processor Information")[0].Properties[k].Val); !ok {
			t.Fatal()
		}
	}
}

func TestParseTestOnBoardDevices(t *testing.T) {
	dmi, err := parseDMI(sample5)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, dmi.Sections, 1); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, onBoardDevicesTests, 8); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, sectionsFromDMI(dmi, "OnBoardDevices")[0].SubSections, 8); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, sectionsFromDMI(dmi, "OnBoardDevices")[0].Type, TypeOnBoardDevices); !ok {
		t.Fatal()
	}
	for _, v := range onBoardDevicesTests {
		subSection := subSectionsFromDMI(dmi, "OnBoardDevices", v.Title)[0]
		for propertyName, property := range v.Properties {
			foundProperty, ok := subSection.Properties[propertyName]
			if !ok {
				t.Fatal()
			}
			if ok := assert.Equal(t, property.Val, foundProperty.Val); !ok {
				t.Fatal()
			}
			if ok := assert.Equal(t, len(property.Items), len(foundProperty.Items)); !ok {
				t.Fatal()
			}
			for index := range property.Items {
				if ok := assert.Equal(t, property.Items[index], foundProperty.Items[index]); !ok {
					t.Fatal()
				}
			}
		}
	}
}

func TestParseTestOEMSpecificTypeWithIndentedListItem(t *testing.T) {
	dmi, err := parseDMI(sample6)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, dmi.Sections, 1); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, sectionsFromDMI(dmi, "Custom Type 216")[0].SubSections, 1); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, sectionsFromDMI(dmi, "Custom Type 216")[0].Type, Type(216)); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, oemSpecificTypeWithIndentedListItems.Properties, 2); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, subSectionsFromDMI(dmi, "Custom Type 216", "OEM-specific Type")[0].Properties, 2); !ok {
		t.Fatal()
	}
	for k, property := range oemSpecificTypeWithIndentedListItems.Properties {
		foundProperty, ok := subSectionsFromDMI(dmi, "Custom Type 216", "OEM-specific Type")[0].Properties[k]
		if !ok {
			t.Fatal()
		}
		if ok := assert.Equal(t, property.Val, foundProperty.Val); !ok {
			t.Fatal()
		}
		if ok := assert.Equal(t, len(property.Items), len(foundProperty.Items)); !ok {
			t.Fatal()
		}
		for index := range property.Items {
			if ok := assert.Equal(t, property.Items[index], foundProperty.Items[index]); !ok {
				t.Fatal()
			}
		}
	}
}

func TestParseTestMultpleSectionsWithSameTypeStr(t *testing.T) {
	dmi, err := parseDMI(sample7)
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, dmi.Sections, 2); !ok {
		t.Fatal()
	}
	subSections := subSectionsFromDMI(dmi, "MemoryDevice", "Memory Device")
	if ok := assert.Len(t, subSections, 2); !ok {
		t.Fatal()
	}
	if ok := assert.Len(t, multpleSectionsWithSameTypeStr, 2); !ok {
		t.Fatal()
	}
	for index, subSection := range subSections {
		expectedSubSection := multpleSectionsWithSameTypeStr[index]
		if ok := assert.Equal(t, subSection.Title, expectedSubSection.Title); !ok {
			t.Fatal()
		}
		if ok := assert.Equal(t, subSection.Title, "Memory Device"); !ok {
			t.Fatal()
		}
		if ok := assert.Equal(t, len(subSection.Properties), len(expectedSubSection.Properties)); !ok {
			t.Fatal()
		}
		for k, property := range subSection.Properties {
			expectedProperty, ok := expectedSubSection.Properties[k]
			if !ok {
				t.Fatal()
			}
			if ok := assert.Equal(t, expectedProperty.Val, property.Val); !ok {
				t.Fatal()
			}
			if ok := assert.Equal(t, len(expectedProperty.Items), len(property.Items)); !ok {
				t.Fatal()
			}
			for index := range expectedProperty.Items {
				if ok := assert.Equal(t, expectedProperty.Items[index], property.Items[index]); !ok {
					t.Fatal()
				}
			}
		}
	}
}

func TestFullDmiDecodeOutputSample(t *testing.T) {
	dmi, err := parseDMI("# dmidecode 3.1\nGetting SMBIOS data from sysfs.\nSMBIOS 3.0.0 present.\nTable at 0x7AEAA000.\n\nHandle 0x0000, DMI type 0, 24 bytes\nBIOS Information\n	Vendor: Intel Corp.\n	Version: BNKBL357.86A.0062.2018.0222.1644\n	Release Date: 02/22/2018\n	Address: 0xF0000\n	Runtime Size: 64 kB\n	ROM Size: 8192 kB\n	Characteristics:\n		PCI is supported\n		BIOS is upgradeable\n		BIOS shadowing is allowed\n		Boot from CD is supported\n		Selectable boot is supported\n		BIOS ROM is socketed\n		EDD is supported\n		5.25\"/1.2 MB floppy services are supported (int 13h)\n		3.5\"/720 kB floppy services are supported (int 13h)\n		3.5\"/2.88 MB floppy services are supported (int 13h)\n		Print screen service is supported (int 5h)\n		Serial services are supported (int 14h)\n		Printer services are supported (int 17h)\n		ACPI is supported\n		USB legacy is supported\n		BIOS boot specification is supported\n		Targeted content distribution is supported\n		UEFI is supported\n	BIOS Revision: 5.6\n	Firmware Revision: 8.12\n\nHandle 0x0001, DMI type 1, 27 bytes\nSystem Information\n	Manufacturer: Intel Corporation\n	Product Name: NUC7i5BNH\n	Version: J31169-311\n	Serial Number: G6BN81700BFU\n	UUID: BD11849D-5ADA-18C3-17BF-94C6911EC515\n	Wake-up Type: Power Switch\n	SKU Number:                                  \n	Family: Intel NUC\n\nHandle 0x0002, DMI type 2, 15 bytes\nBase Board Information\n	Manufacturer: Intel Corporation\n	Product Name: NUC7i5BNB\n	Version: J31144-310\n	Serial Number: GEBN816011HA\n	Asset Tag:                                  \n	Features:\n		Board is a hosting board\n		Board is replaceable\n	Location In Chassis: Default string\n	Chassis Handle: 0x0003\n	Type: Motherboard\n	Contained Object Handles: 0\n\nHandle 0x0003, DMI type 3, 22 bytes\nChassis Information\n	Manufacturer: Intel Corporation\n	Type: Desktop\n	Lock: Not Present\n	Version: 2\n	Serial Number:                                  \n	Asset Tag:                                  \n	Boot-up State: Safe\n	Power Supply State: Safe\n	Thermal State: Safe\n	Security Status: None\n	OEM Information: 0x00000000\n	Height: Unspecified\n	Number Of Power Cords: 1\n	Contained Elements: 0\n	SKU Number:                                  \n\nHandle 0x0004, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J3A1\n	Internal Connector Type: None\n	External Reference Designator: USB1\n	External Connector Type: Access Bus (USB)\n	Port Type: USB\n\nHandle 0x0005, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J3A1\n	Internal Connector Type: None\n	External Reference Designator: USB3\n	External Connector Type: Access Bus (USB)\n	Port Type: USB\n\nHandle 0x0006, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J5A1\n	Internal Connector Type: None\n	External Reference Designator: LAN\n	External Connector Type: RJ-45\n	Port Type: Network Port\n\nHandle 0x0007, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J5A1\n	Internal Connector Type: None\n	External Reference Designator: USB4\n	External Connector Type: Access Bus (USB)\n	Port Type: USB\n\nHandle 0x0008, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J5A1\n	Internal Connector Type: None\n	External Reference Designator: USB5\n	External Connector Type: Access Bus (USB)\n	Port Type: USB\n\nHandle 0x0009, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J9C1 - PCIE DOCKING CONN\n	Internal Connector Type: Other\n	External Reference Designator: Not Specified\n	External Connector Type: None\n	Port Type: Other\n\nHandle 0x000A, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J2B3 - CPU FAN\n	Internal Connector Type: Other\n	External Reference Designator: Not Specified\n	External Connector Type: None\n	Port Type: Other\n\nHandle 0x000B, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J6C2 - EXT HDMI\n	Internal Connector Type: Other\n	External Reference Designator: Not Specified\n	External Connector Type: None\n	Port Type: Other\n\nHandle 0x000C, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J2G1 - GFX VID\n	Internal Connector Type: Other\n	External Reference Designator: Not Specified\n	External Connector Type: None\n	Port Type: Other\n\nHandle 0x000D, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J1G6 - AC JACK\n	Internal Connector Type: Other\n	External Reference Designator: Not Specified\n	External Connector Type: None\n	Port Type: Other\n\nHandle 0x000E, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J7H2 - SATA PWR\n	Internal Connector Type: Other\n	External Reference Designator: Not Specified\n	External Connector Type: None\n	Port Type: Other\n\nHandle 0x000F, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: J6B2\n	Type: x16 PCI Express\n	Current Usage: In Use\n	Length: Long\n	ID: 0\n	Characteristics:\n		3.3 V is provided\n		Opening is shared\n		PME signal is supported\n	Bus Address: 0000:00:01.0\n\nHandle 0x0010, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: J6B1\n	Type: x1 PCI Express\n	Current Usage: In Use\n	Length: Short\n	ID: 1\n	Characteristics:\n		3.3 V is provided\n		Opening is shared\n		PME signal is supported\n	Bus Address: 0000:00:1c.3\n\nHandle 0x0011, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: J6D1\n	Type: x1 PCI Express\n	Current Usage: In Use\n	Length: Short\n	ID: 2\n	Characteristics:\n		3.3 V is provided\n		Opening is shared\n		PME signal is supported\n	Bus Address: 0000:00:1c.4\n\nHandle 0x0012, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: J7B1\n	Type: x1 PCI Express\n	Current Usage: In Use\n	Length: Short\n	ID: 3\n	Characteristics:\n		3.3 V is provided\n		Opening is shared\n		PME signal is supported\n	Bus Address: 0000:00:1c.5\n\nHandle 0x0013, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: J8B4\n	Type: x1 PCI Express\n	Current Usage: In Use\n	Length: Short\n	ID: 4\n	Characteristics:\n		3.3 V is provided\n		Opening is shared\n		PME signal is supported\n	Bus Address: 0000:00:1c.6\n\nHandle 0x0014, DMI type 10, 20 bytes\nOn Board Device 1 Information\n	Type: Video\n	Status: Enabled\n	Description:  Intel(R) HD Graphics Device\nOn Board Device 2 Information\n	Type: Ethernet\n	Status: Enabled\n	Description:  Intel(R) I219-V Gigabit Network Device\nOn Board Device 3 Information\n	Type: Sound\n	Status: Enabled\n	Description:  Realtek High Definition Audio Device\nOn Board Device 4 Information\n	Type: Other\n	Status: Enabled\n	Description: CIR Device\nOn Board Device 5 Information\n	Type: Other\n	Status: Enabled\n	Description: SD\nOn Board Device 6 Information\n	Type: Other\n	Status: Enabled\n	Description: Intel Dual Band Wireless-AC 8265\nOn Board Device 7 Information\n	Type: Other\n	Status: Enabled\n	Description: Bluetooth\nOn Board Device 8 Information\n	Type: Other\n	Status: Disabled\n	Description: Thunderbolt\n\nHandle 0x0015, DMI type 11, 5 bytes\nOEM Strings\n	String 1: Default string\n\nHandle 0x0016, DMI type 12, 5 bytes\nSystem Configuration Options\n	Option 1: Default string\n\nHandle 0x0017, DMI type 32, 20 bytes\nSystem Boot Information\n	Status: No errors detected\n\nHandle 0x0018, DMI type 34, 11 bytes\nManagement Device\n	Description: LM78-1\n	Type: LM78\n	Address: 0x00000000\n	Address Type: I/O Port\n\nHandle 0x0019, DMI type 26, 22 bytes\nVoltage Probe\n	Description: LM78A\n	Location: Motherboard\n	Status: OK\n	Maximum Value: Unknown\n	Minimum Value: Unknown\n	Resolution: Unknown\n	Tolerance: Unknown\n	Accuracy: Unknown\n	OEM-specific Information: 0x00000000\n	Nominal Value: Unknown\n\nHandle 0x001A, DMI type 36, 16 bytes\nManagement Device Threshold Data\n	Lower Non-critical Threshold: 1\n	Upper Non-critical Threshold: 2\n	Lower Critical Threshold: 3\n	Upper Critical Threshold: 4\n	Lower Non-recoverable Threshold: 5\n	Upper Non-recoverable Threshold: 6\n\nHandle 0x001B, DMI type 35, 11 bytes\nManagement Device Component\n	Description: Default string\n	Management Device Handle: 0x0018\n	Component Handle: 0x0019\n	Threshold Handle: 0x001A\n\nHandle 0x001C, DMI type 28, 22 bytes\nTemperature Probe\n	Description: LM78A\n	Location: Motherboard\n	Status: OK\n	Maximum Value: Unknown\n	Minimum Value: Unknown\n	Resolution: Unknown\n	Tolerance: Unknown\n	Accuracy: Unknown\n	OEM-specific Information: 0x00000000\n	Nominal Value: Unknown\n\nHandle 0x001D, DMI type 36, 16 bytes\nManagement Device Threshold Data\n	Lower Non-critical Threshold: 1\n	Upper Non-critical Threshold: 2\n	Lower Critical Threshold: 3\n	Upper Critical Threshold: 4\n	Lower Non-recoverable Threshold: 5\n	Upper Non-recoverable Threshold: 6\n\nHandle 0x001E, DMI type 35, 11 bytes\nManagement Device Component\n	Description: Default string\n	Management Device Handle: 0x0018\n	Component Handle: 0x001C\n	Threshold Handle: 0x001D\n\nHandle 0x001F, DMI type 27, 15 bytes\nCooling Device\n	Temperature Probe Handle: 0x001C\n	Type: Power Supply Fan\n	Status: OK\n	Cooling Unit Group: 1\n	OEM-specific Information: 0x00000000\n	Nominal Speed: Unknown Or Non-rotating\n	Description: Cooling Dev 1\n\nHandle 0x0020, DMI type 36, 16 bytes\nManagement Device Threshold Data\n	Lower Non-critical Threshold: 1\n	Upper Non-critical Threshold: 2\n	Lower Critical Threshold: 3\n	Upper Critical Threshold: 4\n	Lower Non-recoverable Threshold: 5\n	Upper Non-recoverable Threshold: 6\n\nHandle 0x0021, DMI type 35, 11 bytes\nManagement Device Component\n	Description: Default string\n	Management Device Handle: 0x0018\n	Component Handle: 0x001F\n	Threshold Handle: 0x0020\n\nHandle 0x0022, DMI type 27, 15 bytes\nCooling Device\n	Temperature Probe Handle: 0x001C\n	Type: Power Supply Fan\n	Status: OK\n	Cooling Unit Group: 1\n	OEM-specific Information: 0x00000000\n	Nominal Speed: Unknown Or Non-rotating\n	Description: Not Specified\n\nHandle 0x0023, DMI type 36, 16 bytes\nManagement Device Threshold Data\n	Lower Non-critical Threshold: 1\n	Upper Non-critical Threshold: 2\n	Lower Critical Threshold: 3\n	Upper Critical Threshold: 4\n	Lower Non-recoverable Threshold: 5\n	Upper Non-recoverable Threshold: 6\n\nHandle 0x0024, DMI type 35, 11 bytes\nManagement Device Component\n	Description: Default string\n	Management Device Handle: 0x0018\n	Component Handle: 0x0022\n	Threshold Handle: 0x0023\n\nHandle 0x0025, DMI type 29, 22 bytes\nElectrical Current Probe\n	Description: ABC\n	Location: Motherboard\n	Status: OK\n	Maximum Value: Unknown\n	Minimum Value: Unknown\n	Resolution: Unknown\n	Tolerance: Unknown\n	Accuracy: Unknown\n	OEM-specific Information: 0x00000000\n	Nominal Value: Unknown\n\nHandle 0x0026, DMI type 36, 16 bytes\nManagement Device Threshold Data\n\nHandle 0x0027, DMI type 35, 11 bytes\nManagement Device Component\n	Description: Default string\n	Management Device Handle: 0x0018\n	Component Handle: 0x0025\n	Threshold Handle: 0x0026\n\nHandle 0x0028, DMI type 26, 22 bytes\nVoltage Probe\n	Description: LM78A\n	Location: Power Unit\n	Status: OK\n	Maximum Value: Unknown\n	Minimum Value: Unknown\n	Resolution: Unknown\n	Tolerance: Unknown\n	Accuracy: Unknown\n	OEM-specific Information: 0x00000000\n	Nominal Value: Unknown\n\nHandle 0x0029, DMI type 28, 22 bytes\nTemperature Probe\n	Description: LM78A\n	Location: Power Unit\n	Status: OK\n	Maximum Value: Unknown\n	Minimum Value: Unknown\n	Resolution: Unknown\n	Tolerance: Unknown\n	Accuracy: Unknown\n	OEM-specific Information: 0x00000000\n	Nominal Value: Unknown\n\nHandle 0x002A, DMI type 27, 15 bytes\nCooling Device\n	Temperature Probe Handle: 0x0029\n	Type: Power Supply Fan\n	Status: OK\n	Cooling Unit Group: 1\n	OEM-specific Information: 0x00000000\n	Nominal Speed: Unknown Or Non-rotating\n	Description: Cooling Dev 1\n\nHandle 0x002B, DMI type 29, 22 bytes\nElectrical Current Probe\n	Description: ABC\n	Location: Power Unit\n	Status: OK\n	Maximum Value: Unknown\n	Minimum Value: Unknown\n	Resolution: Unknown\n	Tolerance: Unknown\n	Accuracy: Unknown\n	OEM-specific Information: 0x00000000\n	Nominal Value: Unknown\n\nHandle 0x002C, DMI type 39, 22 bytes\nSystem Power Supply\n	Power Unit Group: 1\n	Location: To Be Filled By O.E.M.\n	Name: To Be Filled By O.E.M.\n	Manufacturer: To Be Filled By O.E.M.\n	Serial Number: To Be Filled By O.E.M.\n	Asset Tag: To Be Filled By O.E.M.\n	Model Part Number: To Be Filled By O.E.M.\n	Revision: To Be Filled By O.E.M.\n	Max Power Capacity: Unknown\n	Status: Present, OK\n	Type: Switching\n	Input Voltage Range Switching: Auto-switch\n	Plugged: Yes\n	Hot Replaceable: No\n	Input Voltage Probe Handle: 0x0028\n	Cooling Device Handle: 0x002A\n	Input Current Probe Handle: 0x002B\n\nHandle 0x002D, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  CPU\n	Type: Video\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 0000:00:02.0\n\nHandle 0x002E, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  LAN\n	Type: Ethernet\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 0000:00:1f.6\n\nHandle 0x002F, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  AUDIO\n	Type: Sound\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 00ff:00:1f.7\n\nHandle 0x0030, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  CIR Device\n	Type: Other\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 00ff:00:1f.7\n\nHandle 0x0031, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  SD\n	Type: Other\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 0000:00:00.0\n\nHandle 0x0032, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  Intel Dual Band\n	Type: Other\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 0000:00:00.0\n\nHandle 0x0033, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  Bluetooth\n	Type: Other\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 00ff:00:1f.7\n\nHandle 0x0034, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation:  Thunderbolt\n	Type: Other\n	Status: Disabled\n	Type Instance: 1\n	Bus Address: 00ff:00:1f.7\n\nHandle 0x0035, DMI type 16, 23 bytes\nPhysical Memory Array\n	Location: System Board Or Motherboard\n	Use: System Memory\n	Error Correction Type: None\n	Maximum Capacity: 32 GB\n	Error Information Handle: Not Provided\n	Number Of Devices: 2\n\nHandle 0x0036, DMI type 17, 40 bytes\nMemory Device\n	Array Handle: 0x0035\n	Error Information Handle: Not Provided\n	Total Width: 64 bits\n	Data Width: 64 bits\n	Size: 8192 MB\n	Form Factor: SODIMM\n	Set: None\n	Locator: ChannelA-DIMM0\n	Bank Locator: BANK 0\n	Type: DDR4\n	Type Detail: Synchronous Unbuffered (Unregistered)\n	Speed: 2400 MT/s\n	Manufacturer: 859B\n	Serial Number: E0AD159C\n	Asset Tag: 9876543210\n	Part Number: CT8G4SFD824A.M16FB  \n	Rank: 2\n	Configured Clock Speed: 2133 MT/s\n	Minimum Voltage: 1.2 V\n	Maximum Voltage: 1.2 V\n	Configured Voltage: 1.2 V\n\nHandle 0x0037, DMI type 17, 40 bytes\nMemory Device\n	Array Handle: 0x0035\n	Error Information Handle: Not Provided\n	Total Width: 64 bits\n	Data Width: 64 bits\n	Size: 8192 MB\n	Form Factor: SODIMM\n	Set: None\n	Locator: ChannelB-DIMM0\n	Bank Locator: BANK 2\n	Type: DDR4\n	Type Detail: Synchronous Unbuffered (Unregistered)\n	Speed: 2400 MT/s\n	Manufacturer: 859B\n	Serial Number: E0AD159D\n	Asset Tag: 9876543210\n	Part Number: CT8G4SFD824A.M16FB  \n	Rank: 2\n	Configured Clock Speed: 2133 MT/s\n	Minimum Voltage: 1.2 V\n	Maximum Voltage: 1.2 V\n	Configured Voltage: 1.2 V\n\nHandle 0x0038, DMI type 19, 31 bytes\nMemory Array Mapped Address\n	Starting Address: 0x00000000000\n	Ending Address: 0x003FFFFFFFF\n	Range Size: 16 GB\n	Physical Array Handle: 0x0035\n	Partition Width: 2\n\nHandle 0x0039, DMI type 43, 31 bytes\nTPM Device\n	Vendor ID: CTNI\n	Specification Version: 2.0	Firmware Revision: 11.1\n	Description: INTEL	Characteristics:\n		Family configurable via platform software support\n	OEM-specific Information: 0x00000000\n\nHandle 0x003A, DMI type 7, 19 bytes\nCache Information\n	Socket Designation: L1 Cache\n	Configuration: Enabled, Not Socketed, Level 1\n	Operational Mode: Write Back\n	Location: Internal\n	Installed Size: 128 kB\n	Maximum Size: 128 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Parity\n	System Type: Unified\n	Associativity: 8-way Set-associative\n\nHandle 0x003B, DMI type 7, 19 bytes\nCache Information\n	Socket Designation: L2 Cache\n	Configuration: Enabled, Not Socketed, Level 2\n	Operational Mode: Write Back\n	Location: Internal\n	Installed Size: 512 kB\n	Maximum Size: 512 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Single-bit ECC\n	System Type: Unified\n	Associativity: 4-way Set-associative\n\nHandle 0x003C, DMI type 7, 19 bytes\nCache Information\n	Socket Designation: L3 Cache\n	Configuration: Enabled, Not Socketed, Level 3\n	Operational Mode: Write Back\n	Location: Internal\n	Installed Size: 4096 kB\n	Maximum Size: 4096 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Multi-bit ECC\n	System Type: Unified\n	Associativity: 16-way Set-associative\n\nHandle 0x003D, DMI type 4, 48 bytes\nProcessor Information\n	Socket Designation: U3E1\n	Type: Central Processor\n	Family: Core i5\n	Manufacturer: Intel(R) Corporation\n	ID: E9 06 08 00 FF FB EB BF\n	Signature: Type 0, Family 6, Model 142, Stepping 9\n	Flags:\n		FPU (Floating-point unit on-chip)\n		VME (Virtual mode extension)\n		DE (Debugging extension)\n		PSE (Page size extension)\n		TSC (Time stamp counter)\n		MSR (Model specific registers)\n		PAE (Physical address extension)\n		MCE (Machine check exception)\n		CX8 (CMPXCHG8 instruction supported)\n		APIC (On-chip APIC hardware supported)\n		SEP (Fast system call)\n		MTRR (Memory type range registers)\n		PGE (Page global enable)\n		MCA (Machine check architecture)\n		CMOV (Conditional move instruction supported)\n		PAT (Page attribute table)\n		PSE-36 (36-bit page size extension)\n		CLFSH (CLFLUSH instruction supported)\n		DS (Debug store)\n		ACPI (ACPI supported)\n		MMX (MMX technology supported)\n		FXSR (FXSAVE and FXSTOR instructions supported)\n		SSE (Streaming SIMD extensions)\n		SSE2 (Streaming SIMD extensions 2)\n		SS (Self-snoop)\n		HTT (Multi-threading)\n		TM (Thermal monitor supported)\n		PBE (Pending break enabled)\n	Version: Intel(R) Core(TM) i5-7260U CPU @ 2.20GHz\n	Voltage: 0.8 V\n	External Clock: 100 MHz\n	Max Speed: 2400 MHz\n	Current Speed: 2200 MHz\n	Status: Populated, Enabled\n	Upgrade: Socket BGA1356\n	L1 Cache Handle: 0x003A\n	L2 Cache Handle: 0x003B\n	L3 Cache Handle: 0x003C\n	Serial Number: To Be Filled By O.E.M.\n	Asset Tag: To Be Filled By O.E.M.\n	Part Number: To Be Filled By O.E.M.\n	Core Count: 2\n	Core Enabled: 2\n	Thread Count: 4\n	Characteristics:\n		64-bit capable\n		Multi-Core\n		Hardware Thread\n		Execute Protection\n		Enhanced Virtualization\n		Power/Performance Control\n\nHandle 0x003E, DMI type 20, 35 bytes\nMemory Device Mapped Address\n	Starting Address: 0x00000000000\n	Ending Address: 0x001FFFFFFFF\n	Range Size: 8 GB\n	Physical Device Handle: 0x0036\n	Memory Array Mapped Address Handle: 0x0038\n	Partition Row Position: Unknown\n	Interleave Position: 1\n	Interleaved Data Depth: 1\n\nHandle 0x003F, DMI type 20, 35 bytes\nMemory Device Mapped Address\n	Starting Address: 0x00200000000\n	Ending Address: 0x003FFFFFFFF\n	Range Size: 8 GB\n	Physical Device Handle: 0x0037\n	Memory Array Mapped Address Handle: 0x0038\n	Partition Row Position: Unknown\n	Interleave Position: 2\n	Interleaved Data Depth: 1\n\nHandle 0x0040, DMI type 130, 20 bytes\nOEM-specific Type\n	Header and Data:\n		82 14 40 00 24 41 4D 54 00 00 00 00 00 A5 AF 02\n		C0 00 00 00\n\nHandle 0x0041, DMI type 131, 64 bytes\nOEM-specific Type\n	Header and Data:\n		83 40 41 00 31 00 00 00 00 00 00 00 00 00 00 00\n		F8 00 4E 9D 00 00 00 00 01 00 00 00 08 00 0B 00\n		61 0D 32 00 00 00 00 00 FE 00 D8 15 00 00 00 00\n		00 00 00 00 22 00 00 00 76 50 72 6F 00 00 00 00\n\nHandle 0x0042, DMI type 221, 33 bytes\nOEM-specific Type\n	Header and Data:\n		DD 21 42 00 04 01 00 02 08 01 00 00 02 00 00 00\n		00 84 00 03 00 00 05 00 00 00 04 00 FF FF FF FF\n		FF\n	Strings:\n		Reference Code - CPU\n		uCode Version\n		TXT ACM Version\n		BIOS Guard Version\n\nHandle 0x0043, DMI type 221, 26 bytes\nOEM-specific Type\n	Header and Data:\n		DD 1A 43 00 03 01 00 02 08 01 00 00 02 00 00 00\n		00 00 00 03 04 0B 08 32 61 0D\n	Strings:\n		Reference Code - ME 11.0\n		MEBx version\n		ME Firmware Version\n		Consumer SKU\n\nHandle 0x0044, DMI type 221, 75 bytes\nOEM-specific Type\n	Header and Data:\n		DD 4B 44 00 0A 01 00 02 08 01 00 00 02 03 FF FF\n		FF FF FF 04 00 FF FF FF 21 00 05 00 FF FF FF 21\n		00 06 00 FF FF FF FF FF 07 00 3E 00 00 00 00 08\n		00 34 00 00 00 00 09 00 0B 00 00 00 00 0A 00 3E\n		00 00 00 00 0B 00 34 00 00 00 00\n	Strings:\n		Reference Code - SKL PCH\n		PCH-CRID Status\n		Disabled\n		PCH-CRID Original Value\n		PCH-CRID New Value\n		OPROM - RST - RAID\n		SKL PCH H Bx Hsio Version\n		SKL PCH H Dx Hsio Version\n		KBL PCH H Ax Hsio Version\n		SKL PCH LP Bx Hsio Version\n		SKL PCH LP Cx Hsio Version\n\nHandle 0x0045, DMI type 221, 54 bytes\nOEM-specific Type\n	Header and Data:\n		DD 36 45 00 07 01 00 02 08 01 00 00 02 00 02 08\n		01 00 00 03 00 02 08 01 00 00 04 05 FF FF FF FF\n		FF 06 00 FF FF FF 03 00 07 00 FF FF FF 03 00 08\n		00 FF FF FF FF FF\n	Strings:\n		Reference Code - SA - System Agent\n		Reference Code - MRC\n		SA - PCIe Version\n		SA-CRID Status\n		Disabled\n		SA-CRID Original Value\n		SA-CRID New Value\n		OPROM - VBIOS\n\nHandle 0x0046, DMI type 221, 96 bytes\nOEM-specific Type\n	Header and Data:\n		DD 60 46 00 0D 01 00 00 00 00 A6 00 02 00 FF FF\n		FF FF FF 03 04 FF FF FF FF FF 05 06 FF FF FF FF\n		FF 07 08 FF FF FF FF FF 09 00 00 00 00 00 00 0A\n		00 FF FF FF FF 00 0B 00 08 0C 00 00 00 0C 00 00\n		09 00 66 10 0D 00 FF FF FF FF FF 0E 00 FF FF FF\n		FF FF 0F 10 01 03 04 01 01 11 00 00 07 03 00 00\n	Strings:\n		Lan Phy Version\n		Sensor Firmware Version\n		Debug Mode Status\n		Disabled\n		Performance Mode Status\n		Disabled\n		Debug Use USB(Disabled:Serial)\n		Disabled\n		ICC Overclocking Version\n		UNDI Version\n		EC FW Version\n		GOP Version\n		Base EC FW Version\n		EC-EC Protocol Version\n		Royal Park Version\n		BP1.3.4.1_RP01\n		Platform Version\n\nHandle 0x0047, DMI type 136, 6 bytes\nOEM-specific Type\n	Header and Data:\n		88 06 47 00 00 00\n\nHandle 0x0048, DMI type 14, 20 bytes\nGroup Associations\n	Name: Firmware Version Info\n	Items: 5\n		0x0042 (OEM-specific)\n		0x0043 (OEM-specific)\n		0x0044 (OEM-specific)\n		0x0045 (OEM-specific)\n		0x0046 (OEM-specific)\n\nHandle 0x0049, DMI type 14, 8 bytes\nGroup Associations\n	Name: $MEI\n	Items: 1\n		0x0000 (OEM-specific)\n\nHandle 0x004A, DMI type 219, 81 bytes\nOEM-specific Type\n	Header and Data:\n		DB 51 4A 00 01 03 01 45 02 00 90 06 01 00 66 20\n		00 00 00 00 40 08 00 00 00 00 00 00 00 00 40 02\n		FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF FF\n		FF FF FF FF FF FF FF FF 03 00 00 00 80 00 00 00\n		00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n		00\n	Strings:\n		MEI1\n		MEI2\n		MEI3\n\nHandle 0x004B, DMI type 127, 4 bytes\nEnd Of Table\n\n")
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}
}

func TestFullDmiDecodeOutputSample2(t *testing.T) {
	dmi, err := parseDMI("# dmidecode 3.1\nGetting SMBIOS data from sysfs.\nSMBIOS 3.1 present.\n188 structures occupying 9476 bytes.\nTable at 0x74921000.\n\nHandle 0x0000, DMI type 194, 5 bytes\nOEM-specific Type\n	Header and Data:\n		C2 05 00 00 11\n\nHandle 0x0001, DMI type 199, 40 bytes\nOEM-specific Type\n	Header and Data:\n		C7 28 01 00 2B 00 00 80 16 20 08 02 51 06 05 00\n		34 00 00 80 16 20 02 12 52 06 05 00 50 00 00 02\n		18 20 09 08 54 06 05 00\n\nHandle 0x0002, DMI type 201, 16 bytes\nOEM-specific Type\n	Header and Data:\n		C9 10 02 00 10 02 00 00 40 0D 01 00 0E 00 00 80\n\nHandle 0x0003, DMI type 0, 26 bytes\nBIOS Information\n	Vendor: HPE\n	Version: U40\n	Release Date: 10/02/2018\n	Address: 0xF0000\n	Runtime Size: 64 kB\n	ROM Size: 64 MB\n	Characteristics:\n		PCI is supported\n		PNP is supported\n		BIOS is upgradeable\n		BIOS shadowing is allowed\n		ESCD support is available\n		Boot from CD is supported\n		Selectable boot is supported\n		EDD is supported\n		5.25\"/360 kB floppy services are supported (int 13h)\n		5.25\"/1.2 MB floppy services are supported (int 13h)\n		3.5\"/720 kB floppy services are supported (int 13h)\n		Print screen service is supported (int 5h)\n		8042 keyboard services are supported (int 9h)\n		Serial services are supported (int 14h)\n		Printer services are supported (int 17h)\n		CGA/mono video services are supported (int 10h)\n		ACPI is supported\n		USB legacy is supported\n		BIOS boot specification is supported\n		Function key-initiated network boot is supported\n		Targeted content distribution is supported\n		UEFI is supported\n	BIOS Revision: 1.46\n	Firmware Revision: 1.37\n\nHandle 0x0004, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J90041\n	Internal Connector Type: Access Bus (USB)\n	External Reference Designator: Front USB key\n	External Connector Type: Access Bus (USB)\n	Port Type: USB\n\nHandle 0x0005, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J90043\n	Internal Connector Type: Access Bus (USB)\n	External Reference Designator: Front USB key\n	External Connector Type: Access Bus (USB)\n	Port Type: USB\n\nHandle 0x0006, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J161\n	Internal Connector Type: Access Bus (USB)\n	External Reference Designator: Internal USB key\n	External Connector Type: Access Bus (USB)\n	Port Type: USB\n\nHandle 0x0007, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J90043\n	Internal Connector Type: None\n	External Reference Designator: Video PORT\n	External Connector Type: DB-15 female\n	Port Type: Video Port\n\nHandle 0x0008, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J28\n	Internal Connector Type: None\n	External Reference Designator: Video PORT\n	External Connector Type: DB-15 female\n	Port Type: Video Port\n\nHandle 0x0009, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J39\n	Internal Connector Type: None\n	External Reference Designator: ILO NIC PORT\n	External Connector Type: RJ-45\n	Port Type: Network Port\n\nHandle 0x000A, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J109\n	Internal Connector Type: None\n	External Reference Designator: NIC PORT 1\n	External Connector Type: RJ-45\n	Port Type: Network Port\n\nHandle 0x000B, DMI type 8, 9 bytes\nPort Connector Information\n	Internal Reference Designator: J109\n	Internal Connector Type: None\n	External Reference Designator: NIC PORT 2\n	External Connector Type: RJ-45\n	Port Type: Network Port\n\nHandle 0x000C, DMI type 16, 23 bytes\nPhysical Memory Array\n	Location: System Board Or Motherboard\n	Use: System Memory\n	Error Correction Type: Multi-bit ECC\n	Maximum Capacity: 2 TB\n	Error Information Handle: Not Provided\n	Number Of Devices: 8\n\nHandle 0x000D, DMI type 16, 23 bytes\nPhysical Memory Array\n	Location: System Board Or Motherboard\n	Use: System Memory\n	Error Correction Type: Multi-bit ECC\n	Maximum Capacity: 2 TB\n	Error Information Handle: Not Provided\n	Number Of Devices: 8\n\nHandle 0x000E, DMI type 38, 18 bytes\nIPMI Device Information\n	Interface Type: KCS (Keyboard Control Style)\n	Specification Version: 2.0\n	I2C Slave Address: 0x10\n	NV Storage Device: Not Present\n	Base Address: 0x0000000000000CA2 (I/O)\n	Register Spacing: Successive Byte Boundaries\n\nHandle 0x000F, DMI type 193, 9 bytes\nOEM-specific Type\n	Header and Data:\n		C1 09 0F 00 01 01 00 02 03\n	Strings:\n		v1.46 (10/02/2018)\n		         \n		          \n\nHandle 0x0010, DMI type 195, 7 bytes\nOEM-specific Type\n	Header and Data:\n		C3 07 10 00 01 0F 02\n	Strings:\n		$0E110863\n\nHandle 0x0011, DMI type 198, 14 bytes\nOEM-specific Type\n	Header and Data:\n		C6 0E 11 00 01 00 00 00 00 00 01 0A FF FF\n\nHandle 0x0012, DMI type 215, 6 bytes\nOEM-specific Type\n	Header and Data:\n		D7 06 12 00 00 05\n\nHandle 0x0013, DMI type 223, 11 bytes\nOEM-specific Type\n	Header and Data:\n		DF 0B 13 00 66 46 70 00 00 00 00\n\nHandle 0x0014, DMI type 236, 21 bytes\nOEM-specific Type\n	Header and Data:\n		EC 15 14 00 A0 00 00 EC 00 00 00 00 00 00 00 00\n		00 04 04 00 01\n	Strings:\n		Gen10 1x2 SFF CB3\n\nHandle 0x0015, DMI type 236, 21 bytes\nOEM-specific Type\n	Header and Data:\n		EC 15 15 00 A0 00 00 EC 00 00 00 00 00 00 00 00\n		00 02 02 00 01\n	Strings:\n		GEN10 1x2 SFF CB3\n\nHandle 0x0016, DMI type 236, 21 bytes\nOEM-specific Type\n	Header and Data:\n		EC 15 16 00 A0 01 00 EC 00 00 00 00 00 00 00 00\n		00 02 02 00 01\n	Strings:\n		GEN10 1x2 SFF CB3\n\nHandle 0x0017, DMI type 19, 31 bytes\nMemory Array Mapped Address\n	Starting Address: 0x00000000000\n	Ending Address: 0x000BFFFFFFF\n	Range Size: 3 GB\n	Physical Array Handle: 0x000C\n	Partition Width: 1\n\nHandle 0x0018, DMI type 19, 31 bytes\nMemory Array Mapped Address\n	Starting Address: 0x0000000100000000k\n	Ending Address: 0x000000083FFFFFFFk\n	Range Size: 29 GB\n	Physical Array Handle: 0x000D\n	Partition Width: 1\n\nHandle 0x0019, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: None\n	Locator: PROC 1 DIMM 1\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x001A, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 1\n	Locator: PROC 1 DIMM 2\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x001B, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: 16384 MB\n	Form Factor: DIMM\n	Set: 2\n	Locator: PROC 1 DIMM 3\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous Registered (Buffered)\n	Speed: 2666 MT/s\n	Manufacturer: HPE\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: 840757-091\n	Rank: 1\n	Configured Clock Speed: 2400 MT/s\n	Minimum Voltage: 1.2 V\n	Maximum Voltage: 1.2 V\n	Configured Voltage: 1.2 V\n\nHandle 0x001C, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 3\n	Locator: PROC 1 DIMM 4\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x001D, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 4\n	Locator: PROC 1 DIMM 5\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x001E, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 5\n	Locator: PROC 1 DIMM 6\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x001F, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 6\n	Locator: PROC 1 DIMM 7\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0020, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000C\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 7\n	Locator: PROC 1 DIMM 8\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0021, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 8\n	Locator: PROC 2 DIMM 1\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0022, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 9\n	Locator: PROC 2 DIMM 2\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0023, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: 16384 MB\n	Form Factor: DIMM\n	Set: 10\n	Locator: PROC 2 DIMM 3\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous Registered (Buffered)\n	Speed: 2666 MT/s\n	Manufacturer: HPE\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: 840757-091\n	Rank: 1\n	Configured Clock Speed: 2400 MT/s\n	Minimum Voltage: 1.2 V\n	Maximum Voltage: 1.2 V\n	Configured Voltage: 1.2 V\n\nHandle 0x0024, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 11\n	Locator: PROC 2 DIMM 4\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0025, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 12\n	Locator: PROC 2 DIMM 5\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0026, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 13\n	Locator: PROC 2 DIMM 6\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0027, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 14\n	Locator: PROC 2 DIMM 7\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0028, DMI type 17, 84 bytes\nMemory Device\n	Array Handle: 0x000D\n	Error Information Handle: Not Provided\n	Total Width: 72 bits\n	Data Width: 64 bits\n	Size: No Module Installed\n	Form Factor: DIMM\n	Set: 15\n	Locator: PROC 2 DIMM 8\n	Bank Locator: Not Specified\n	Type: DDR4\n	Type Detail: Synchronous\n	Speed: Unknown\n	Manufacturer: UNKNOWN\n	Serial Number: Not Specified\n	Asset Tag: Not Specified\n	Part Number: NOT AVAILABLE\n	Rank: Unknown\n	Configured Clock Speed: Unknown\n	Minimum Voltage: Unknown\n	Maximum Voltage: Unknown\n	Configured Voltage: Unknown\n\nHandle 0x0029, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 29 00 19 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x002A, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 2A 00 1A 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x002B, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 2B 00 1B 00 01 02 03\n	Strings:\n		Samsung         \n		M393A2K40CB2-CTD    \n		39E06372\n\nHandle 0x002C, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 2C 00 1C 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x002D, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 2D 00 1D 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x002E, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 2E 00 1E 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x002F, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 2F 00 1F 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0030, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 30 00 20 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0031, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 31 00 21 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0032, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 32 00 22 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0033, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 33 00 23 00 01 02 03\n	Strings:\n		Samsung         \n		M393A2K40CB2-CTD    \n		39E05413\n\nHandle 0x0034, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 34 00 24 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0035, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 35 00 25 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0036, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 36 00 26 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0037, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 37 00 27 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0038, DMI type 237, 9 bytes\nOEM-specific Type\n	Header and Data:\n		ED 09 38 00 28 00 01 02 03\n	Strings:\n		Unknown         \n		NOT AVAILABLE   \n		NOT AVAILABLE   \n\nHandle 0x0039, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 39 00 19 00 10 00 00 00 00 00 00 00\n\nHandle 0x003A, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 3A 00 1A 00 10 00 00 00 00 00 00 00\n\nHandle 0x003B, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 3B 00 1B 00 11 00 00 00 B0 04 B0 04\n\nHandle 0x003C, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 3C 00 1C 00 10 00 00 00 00 00 00 00\n\nHandle 0x003D, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 3D 00 1D 00 10 00 00 00 00 00 00 00\n\nHandle 0x003E, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 3E 00 1E 00 10 00 00 00 00 00 00 00\n\nHandle 0x003F, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 3F 00 1F 00 10 00 00 00 00 00 00 00\n\nHandle 0x0040, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 40 00 20 00 10 00 00 00 00 00 00 00\n\nHandle 0x0041, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 41 00 21 00 10 00 00 00 00 00 00 00\n\nHandle 0x0042, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 42 00 22 00 10 00 00 00 00 00 00 00\n\nHandle 0x0043, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 43 00 23 00 11 00 00 00 B0 04 B0 04\n\nHandle 0x0044, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 44 00 24 00 10 00 00 00 00 00 00 00\n\nHandle 0x0045, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 45 00 25 00 10 00 00 00 00 00 00 00\n\nHandle 0x0046, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 46 00 26 00 10 00 00 00 00 00 00 00\n\nHandle 0x0047, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 47 00 27 00 10 00 00 00 00 00 00 00\n\nHandle 0x0048, DMI type 232, 14 bytes\nOEM-specific Type\n	Header and Data:\n		E8 0E 48 00 28 00 10 00 00 00 00 00 00 00\n\nHandle 0x0049, DMI type 7, 27 bytes\nCache Information\n	Socket Designation: L1-Cache\n	Configuration: Enabled, Not Socketed, Level 1\n	Operational Mode: Write Back\n	Location: Internal\n	Installed Size: 512 kB\n	Maximum Size: 512 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Single-bit ECC\n	System Type: Unified\n	Associativity: 8-way Set-associative\n\nHandle 0x004A, DMI type 7, 27 bytes\nCache Information\n	Socket Designation: L2-Cache\n	Configuration: Enabled, Not Socketed, Level 2\n	Operational Mode: Varies With Memory Address\n	Location: Internal\n	Installed Size: 8192 kB\n	Maximum Size: 8192 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Single-bit ECC\n	System Type: Unified\n	Associativity: 16-way Set-associative\n\nHandle 0x004B, DMI type 7, 27 bytes\nCache Information\n	Socket Designation: L3-Cache\n	Configuration: Enabled, Not Socketed, Level 3\n	Operational Mode: Varies With Memory Address\n	Location: Internal\n	Installed Size: 11264 kB\n	Maximum Size: 11264 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Single-bit ECC\n	System Type: Unified\n	Associativity: Fully Associative\n\nHandle 0x004C, DMI type 4, 48 bytes\nProcessor Information\n	Socket Designation: Proc 1\n	Type: Central Processor\n	Family: Xeon\n	Manufacturer: Intel(R) Corporation\n	ID: 54 06 05 00 FF FB EB BF\n	Signature: Type 0, Family 6, Model 85, Stepping 4\n	Flags:\n		FPU (Floating-point unit on-chip)\n		VME (Virtual mode extension)\n		DE (Debugging extension)\n		PSE (Page size extension)\n		TSC (Time stamp counter)\n		MSR (Model specific registers)\n		PAE (Physical address extension)\n		MCE (Machine check exception)\n		CX8 (CMPXCHG8 instruction supported)\n		APIC (On-chip APIC hardware supported)\n		SEP (Fast system call)\n		MTRR (Memory type range registers)\n		PGE (Page global enable)\n		MCA (Machine check architecture)\n		CMOV (Conditional move instruction supported)\n		PAT (Page attribute table)\n		PSE-36 (36-bit page size extension)\n		CLFSH (CLFLUSH instruction supported)\n		DS (Debug store)\n		ACPI (ACPI supported)\n		MMX (MMX technology supported)\n		FXSR (FXSAVE and FXSTOR instructions supported)\n		SSE (Streaming SIMD extensions)\n		SSE2 (Streaming SIMD extensions 2)\n		SS (Self-snoop)\n		HTT (Multi-threading)\n		TM (Thermal monitor supported)\n		PBE (Pending break enabled)\n	Version: Intel(R) Xeon(R) Silver 4108 CPU @ 1.80GHz\n	Voltage: 1.6 V\n	External Clock: 100 MHz\n	Max Speed: 4000 MHz\n	Current Speed: 1800 MHz\n	Status: Populated, Enabled\n	Upgrade: Socket LGA3647-1\n	L1 Cache Handle: 0x0049\n	L2 Cache Handle: 0x004A\n	L3 Cache Handle: 0x004B\n	Serial Number: Not Specified\n	Asset Tag: UNKNOWN\n	Part Number: Not Specified\n	Core Count: 8\n	Core Enabled: 8\n	Thread Count: 16\n	Characteristics:\n		64-bit capable\n		Multi-Core\n		Hardware Thread\n		Execute Protection\n		Enhanced Virtualization\n		Power/Performance Control\n\nHandle 0x004D, DMI type 7, 27 bytes\nCache Information\n	Socket Designation: L1-Cache\n	Configuration: Enabled, Not Socketed, Level 1\n	Operational Mode: Write Back\n	Location: Internal\n	Installed Size: 512 kB\n	Maximum Size: 512 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Single-bit ECC\n	System Type: Unified\n	Associativity: 8-way Set-associative\n\nHandle 0x004E, DMI type 7, 27 bytes\nCache Information\n	Socket Designation: L2-Cache\n	Configuration: Enabled, Not Socketed, Level 2\n	Operational Mode: Varies With Memory Address\n	Location: Internal\n	Installed Size: 8192 kB\n	Maximum Size: 8192 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Single-bit ECC\n	System Type: Unified\n	Associativity: 16-way Set-associative\n\nHandle 0x004F, DMI type 7, 27 bytes\nCache Information\n	Socket Designation: L3-Cache\n	Configuration: Enabled, Not Socketed, Level 3\n	Operational Mode: Varies With Memory Address\n	Location: Internal\n	Installed Size: 11264 kB\n	Maximum Size: 11264 kB\n	Supported SRAM Types:\n		Synchronous\n	Installed SRAM Type: Synchronous\n	Speed: Unknown\n	Error Correction Type: Single-bit ECC\n	System Type: Unified\n	Associativity: Fully Associative\n\nHandle 0x0050, DMI type 4, 48 bytes\nProcessor Information\n	Socket Designation: Proc 2\n	Type: Central Processor\n	Family: Xeon\n	Manufacturer: Intel(R) Corporation\n	ID: 54 06 05 00 FF FB EB BF\n	Signature: Type 0, Family 6, Model 85, Stepping 4\n	Flags:\n		FPU (Floating-point unit on-chip)\n		VME (Virtual mode extension)\n		DE (Debugging extension)\n		PSE (Page size extension)\n		TSC (Time stamp counter)\n		MSR (Model specific registers)\n		PAE (Physical address extension)\n		MCE (Machine check exception)\n		CX8 (CMPXCHG8 instruction supported)\n		APIC (On-chip APIC hardware supported)\n		SEP (Fast system call)\n		MTRR (Memory type range registers)\n		PGE (Page global enable)\n		MCA (Machine check architecture)\n		CMOV (Conditional move instruction supported)\n		PAT (Page attribute table)\n		PSE-36 (36-bit page size extension)\n		CLFSH (CLFLUSH instruction supported)\n		DS (Debug store)\n		ACPI (ACPI supported)\n		MMX (MMX technology supported)\n		FXSR (FXSAVE and FXSTOR instructions supported)\n		SSE (Streaming SIMD extensions)\n		SSE2 (Streaming SIMD extensions 2)\n		SS (Self-snoop)\n		HTT (Multi-threading)\n		TM (Thermal monitor supported)\n		PBE (Pending break enabled)\n	Version: Intel(R) Xeon(R) Silver 4108 CPU @ 1.80GHz\n	Voltage: 1.6 V\n	External Clock: 100 MHz\n	Max Speed: 4000 MHz\n	Current Speed: 1800 MHz\n	Status: Populated, Enabled\n	Upgrade: Socket LGA3647-1\n	L1 Cache Handle: 0x004D\n	L2 Cache Handle: 0x004E\n	L3 Cache Handle: 0x004F\n	Serial Number: Not Specified\n	Asset Tag: UNKNOWN\n	Part Number: Not Specified\n	Core Count: 8\n	Core Enabled: 8\n	Thread Count: 16\n	Characteristics:\n		64-bit capable\n		Multi-Core\n		Hardware Thread\n		Execute Protection\n		Enhanced Virtualization\n		Power/Performance Control\n\nHandle 0x0051, DMI type 211, 7 bytes\nOEM-specific Type\n	Header and Data:\n		D3 07 51 00 4C 00 0A\n\nHandle 0x0052, DMI type 211, 7 bytes\nOEM-specific Type\n	Header and Data:\n		D3 07 52 00 50 00 0A\n\nHandle 0x0053, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 53 00 4C 00 19 00 09 A8 01 01 FF FF FF FF\n		05 00 00 00 00 00\n\nHandle 0x0054, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 54 00 4C 00 1A 00 09 A4 01 01 FF FF FF FF\n		04 00 00 00 00 00\n\nHandle 0x0055, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 55 00 4C 00 1B 00 09 A0 01 01 FF FF FF FF\n		03 00 00 00 00 00\n\nHandle 0x0056, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 56 00 4C 00 1C 00 09 A2 01 01 FF FF FF FF\n		03 00 00 00 00 00\n\nHandle 0x0057, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 57 00 4C 00 1D 00 0A A2 01 00 FF FF FF FF\n		00 00 00 00 00 00\n\nHandle 0x0058, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 58 00 4C 00 1E 00 0A A0 01 00 FF FF FF FF\n		00 00 00 00 00 00\n\nHandle 0x0059, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 59 00 4C 00 1F 00 0A A4 01 00 FF FF FF FF\n		01 00 00 00 00 00\n\nHandle 0x005A, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 5A 00 4C 00 20 00 0A A8 01 00 FF FF FF FF\n		02 00 00 00 00 00\n\nHandle 0x005B, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 5B 00 50 00 21 00 0B A8 01 03 FF FF FF FF\n		05 00 00 00 00 00\n\nHandle 0x005C, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 5C 00 50 00 22 00 0B A4 01 03 FF FF FF FF\n		04 00 00 00 00 00\n\nHandle 0x005D, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 5D 00 50 00 23 00 0B A0 01 03 FF FF FF FF\n		03 00 00 00 00 00\n\nHandle 0x005E, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 5E 00 50 00 24 00 0B A2 01 03 FF FF FF FF\n		03 00 00 00 00 00\n\nHandle 0x005F, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 5F 00 50 00 25 00 0C A2 01 02 FF FF FF FF\n		00 00 00 00 00 00\n\nHandle 0x0060, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 60 00 50 00 26 00 0C A0 01 02 FF FF FF FF\n		00 00 00 00 00 00\n\nHandle 0x0061, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 61 00 50 00 27 00 0C A4 01 02 FF FF FF FF\n		01 00 00 00 00 00\n\nHandle 0x0062, DMI type 227, 22 bytes\nOEM-specific Type\n	Header and Data:\n		E3 16 62 00 50 00 28 00 0C A8 01 02 FF FF FF FF\n		02 00 00 00 00 00\n\nHandle 0x0063, DMI type 197, 26 bytes\nOEM-specific Type\n	Header and Data:\n		C5 1A 63 00 4C 00 00 05 FF 01 55 00 00 00 00 00\n		4F B0 C6 B9 EE B7 5F 02 80 25\n\nHandle 0x0064, DMI type 197, 26 bytes\nOEM-specific Type\n	Header and Data:\n		C5 1A 64 00 50 00 10 04 FF 02 55 00 00 00 00 00\n		6B 7F 49 29 6E A9 58 02 80 25\n\nHandle 0x0065, DMI type 1, 27 bytes\nSystem Information\n	Manufacturer: HPE\n	Product Name: ProLiant XL450 Gen10\n	Version: Not Specified\n	Serial Number: CZ3904N3MG\n	UUID: 36343638-3532-5A43-3339-30344E334D47\n	Wake-up Type: Power Switch\n	SKU Number: 864625-B21\n	Family: ProLiant\n\nHandle 0x0066, DMI type 226, 21 bytes\nOEM-specific Type\n	Header and Data:\n		E2 15 66 00 38 36 34 36 32 35 43 5A 33 39 30 34\n		4E 33 4D 47 01\n	Strings:\n		CZ3904N3MG\n\nHandle 0x0067, DMI type 39, 22 bytes\nSystem Power Supply\n	Power Unit Group: 1\n	Location: Not Specified\n	Name: Power Supply 1\n	Manufacturer: HPE\n	Serial Number: 5WBXT0C4DB300L\n	Asset Tag: Not Specified\n	Model Part Number: 865414-B21\n	Revision: Not Specified\n	Max Power Capacity: 800 W\n	Status: Present, OK\n	Type: Switching\n	Input Voltage Range Switching: Auto-switch\n	Plugged: Yes\n	Hot Replaceable: Yes\n\nHandle 0x0068, DMI type 39, 22 bytes\nSystem Power Supply\n	Power Unit Group: 1\n	Location: Not Specified\n	Name: Power Supply 2\n	Manufacturer: HPE\n	Serial Number: 5WBXT0C4DB30MM\n	Asset Tag: Not Specified\n	Model Part Number: 865414-B21\n	Revision: Not Specified\n	Max Power Capacity: 800 W\n	Status: Present, OK\n	Type: Switching\n	Input Voltage Range Switching: Auto-switch\n	Plugged: Yes\n	Hot Replaceable: Yes\n\nHandle 0x0069, DMI type 39, 22 bytes\nSystem Power Supply\n	Power Unit Group: 1\n	Location: Not Specified\n	Name: Power Supply 3\n	Manufacturer: HPE\n	Serial Number: 5WBXT0C4DB30OQ\n	Asset Tag: Not Specified\n	Model Part Number: 865414-B21\n	Revision: Not Specified\n	Max Power Capacity: 800 W\n	Status: Present, OK\n	Type: Switching\n	Input Voltage Range Switching: Auto-switch\n	Plugged: Yes\n	Hot Replaceable: Yes\n\nHandle 0x006A, DMI type 39, 22 bytes\nSystem Power Supply\n	Power Unit Group: 1\n	Location: Not Specified\n	Name: Power Supply 4\n	Manufacturer: HPE\n	Serial Number:                 \n	Asset Tag: Not Specified\n	Model Part Number:                                 \n	Revision: Not Specified\n	Max Power Capacity: Unknown\n	Status: Present, Critical\n	Type: Switching\n	Input Voltage Range Switching: Auto-switch\n	Plugged: Yes\n	Hot Replaceable: Yes\n\nHandle 0x006B, DMI type 229, 52 bytes\nOEM-specific Type\n	Header and Data:\n		E5 34 6B 00 24 57 48 45 00 B0 68 8A 00 00 00 00\n		00 10 01 00 24 53 4D 56 D0 FD BE B7 00 00 00 00\n		08 00 00 00 24 5A 58 54 00 A0 68 8A 00 00 00 00\n		A9 00 00 00\n\nHandle 0x006C, DMI type 230, 11 bytes\nOEM-specific Type\n	Header and Data:\n		E6 0B 6C 00 67 00 01 02 03 FF FF\n	Strings:\n		DELTA\n		06\n\nHandle 0x006D, DMI type 230, 11 bytes\nOEM-specific Type\n	Header and Data:\n		E6 0B 6D 00 68 00 01 02 03 FF FF\n	Strings:\n		DELTA\n		06\n\nHandle 0x006E, DMI type 230, 11 bytes\nOEM-specific Type\n	Header and Data:\n		E6 0B 6E 00 69 00 01 02 03 FF FF\n	Strings:\n		DELTA\n		06\n\nHandle 0x006F, DMI type 230, 11 bytes\nOEM-specific Type\n	Header and Data:\n		E6 0B 6F 00 6A 00 00 01 03 FF FF\n	Strings:\n		        \n\nHandle 0x0070, DMI type 219, 32 bytes\nOEM-specific Type\n	Header and Data:\n		DB 20 70 00 CF 33 00 00 1F 00 00 00 00 00 00 00\n		07 98 00 00 00 00 00 00 01 00 00 00 00 00 00 00\n\nHandle 0x0071, DMI type 3, 21 bytes\nChassis Information\n	Manufacturer: HPE\n	Type: Multi-system\n	Lock: Not Present\n	Version: Not Specified\n	Serial Number: CZ3904N3MG\n	Asset Tag:                                 \n	Boot-up State: Safe\n	Power Supply State: Safe\n	Thermal State: Safe\n	Security Status: Unknown\n	OEM Information: 0x00000000\n	Height: 4 U\n	Number Of Power Cords: 4\n	Contained Elements: 0\n\nHandle 0x0072, DMI type 3, 21 bytes\nChassis Information\n	Manufacturer: HPE\n	Type: Multi-system\n	Lock: Not Present\n	Version: Not Specified\n	Serial Number: CZ3904N3LX\n	Asset Tag: 864668-B21\n	Boot-up State: Safe\n	Power Supply State: Safe\n	Thermal State: Safe\n	Security Status: Unknown\n	OEM Information: 0x00000000\n	Height: 4 U\n	Number Of Power Cords: 4\n	Contained Elements: 0\n\nHandle 0x0073, DMI type 11, 5 bytes\nOEM Strings\n	String 1: PSF:                                                            \n	String 2: Product ID: 864625-B21\n	String 3: CPN: Apollo 4500 Chassis\n	String 4: OEM String: \n\nHandle 0x0074, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 74 00 01 00 01 02 07 01 2E 0A 02 E2 07 00\n		00 00 00 00 00 00 00\n	Strings:\n		System ROM\n		v1.46 (10/02/2018)\n\nHandle 0x0075, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 75 00 02 00 01 02 07 01 2E 0A 02 E2 07 00\n		00 00 00 00 00 00 00\n	Strings:\n		Redundant System ROM\n		v1.46 (10/02/2018)\n\nHandle 0x0076, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 76 00 06 00 01 02 05 20 31 2E 33 2E 31 39\n		00 00 00 00 00 00 00\n	Strings:\n		Chassis Firmware\n		 1.3.19\n\nHandle 0x0077, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 77 00 08 00 01 00 01 18 18 00 00 00 00 00\n		00 00 00 00 00 00 00\n	Strings:\n		System Programmable Logic Device\n\nHandle 0x0078, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 78 00 09 00 01 00 0C 04 00 00 00 04 00 89\n		01 00 00 00 00 00 00\n	Strings:\n		Server Platform Services (SPS) Firmware\n\nHandle 0x0079, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 79 00 0C 00 01 02 0A 04 00 00 0A 00 00 00\n		00 00 00 00 00 00 00\n	Strings:\n		Intelligent Platform Abstraction Data\n		4.0.0 Build 10\n\nHandle 0x007A, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 7A 00 0F 00 01 02 06 14 01 00 00 00 00 00\n		00 00 00 00 00 00 00\n	Strings:\n		NVMe Backplane 1 Firmware\n		1.20\n\nHandle 0x007B, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 7B 00 10 00 01 02 09 03 14 9A 00 00 00 00\n		00 00 00 00 00 00 00\n	Strings:\n		Intelligent Provisioning\n		3.20.154\n\nHandle 0x007C, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 7C 00 11 00 01 00 0B 02 00 01 00 00 00 00\n		00 2E 8A C7 74 00 00\n	Strings:\n		ME SPI Descriptor\n\nHandle 0x007D, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 7D 00 12 00 01 00 0C 00 00 01 00 06 00 01\n		00 00 00 DD 9C 00 00\n	Strings:\n		Innovation Engine (IE) Firmware\n\nHandle 0x007E, DMI type 216, 23 bytes\nOEM-specific Type\n	Header and Data:\n		D8 17 7E 00 30 00 01 02 02 25 00 00 00 00 00 00\n		00 00 00 00 00 00 00\n	Strings:\n		Embedded Video Controller\n		2.5\n\nHandle 0x007F, DMI type 2, 15 bytes\nBase Board Information\n	Manufacturer: HPE\n	Product Name: ProLiant XL450 Gen10\n	Version: Not Specified\n	Serial Number: PWAMT%%LMBN03P\n	Asset Tag:                                 \n	Features:\n		Board is a hosting board\n		Board is removable\n		Board is replaceable\n	Location In Chassis: Node 1\n	Chassis Handle: 0x0071\n	Type: Motherboard\n	Contained Object Handles: 0\n\nHandle 0x0080, DMI type 243, 38 bytes\nOEM-specific Type\n	Header and Data:\n		F3 26 80 00 78 00 77 56 4E B3 DC 21 D3 45 87 2B\n		42 F7 6F EE 90 53 47 A4 B1 A6 2A 38 4F 5A 3C 10\n		86 80 0A 00 01 01\n\nHandle 0x0081, DMI type 224, 12 bytes\nOEM-specific Type\n	Header and Data:\n		E0 0C 81 00 00 00 00 01 FE FF 00 00\n\nHandle 0x0082, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation: Embedded LOM 1 Port 1\n	Type: Ethernet\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 0000:02:00.0\n\nHandle 0x0083, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation: Embedded LOM 1 Port 2\n	Type: Ethernet\n	Status: Enabled\n	Type Instance: 2\n	Bus Address: 0000:02:00.1\n\nHandle 0x0084, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation: Embedded FlexibleLOM 1 Port 1\n	Type: Ethernet\n	Status: Enabled\n	Type Instance: 3\n	Bus Address: 0000:37:00.0\n\nHandle 0x0085, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation: Embedded FlexibleLOM 1 Port 2\n	Type: Ethernet\n	Status: Enabled\n	Type Instance: 4\n	Bus Address: 0000:37:00.1\n\nHandle 0x0086, DMI type 41, 11 bytes\nOnboard Device\n	Reference Designation: Embedded Device\n	Type: Video\n	Status: Enabled\n	Type Instance: 1\n	Bus Address: 0000:01:00.1\n\nHandle 0x0087, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: PCI-E Slot 2\n	Type: x8 PCI Express 3\n	Current Usage: In Use\n	Length: Long\n	ID: 2\n	Characteristics:\n		3.3 V is provided\n		PME signal is supported\n	Bus Address: 0000:12:00.0\n\nHandle 0x0088, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: PCI-E Slot 1\n	Type: x8 PCI Express 3\n	Current Usage: Available\n	Length: Long\n	ID: 1\n	Characteristics:\n		3.3 V is provided\n		PME signal is supported\n	Bus Address: 0000:13:00.0\n\nHandle 0x0089, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: NVMe Slot \n	Type: x4 PCI Express 3 SFF-8639\n	Current Usage: Available\n	Length: Other\n	ID: 7\n	Characteristics:\n		3.3 V is provided\n		Hot-plug devices are supported\n		SMBus signal is supported\n	Bus Address: 0000:38:00.0\n\nHandle 0x008A, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: NVMe Slot \n	Type: x4 PCI Express 3 SFF-8639\n	Current Usage: Available\n	Length: Other\n	ID: 8\n	Characteristics:\n		3.3 V is provided\n		Hot-plug devices are supported\n		SMBus signal is supported\n	Bus Address: 0000:39:00.0\n\nHandle 0x008B, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: NVMe Slot \n	Type: x4 PCI Express 3 SFF-8639\n	Current Usage: Available\n	Length: Other\n	ID: 11\n	Characteristics:\n		3.3 V is provided\n		Hot-plug devices are supported\n		SMBus signal is supported\n	Bus Address: 0000:5d:00.0\n\nHandle 0x008C, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: NVMe Slot \n	Type: x4 PCI Express 3 SFF-8639\n	Current Usage: Available\n	Length: Other\n	ID: 12\n	Characteristics:\n		3.3 V is provided\n		Hot-plug devices are supported\n		SMBus signal is supported\n	Bus Address: 0000:5e:00.0\n\nHandle 0x008D, DMI type 9, 17 bytes\nSystem Slot Information\n	Designation: PCI-E Slot 3\n	Type: x16 PCI Express 3\n	Current Usage: Available\n	Length: Long\n	ID: 3\n	Characteristics:\n		3.3 V is provided\n		PME signal is supported\n	Bus Address: 0000:af:00.0\n\nHandle 0x008E, DMI type 32, 11 bytes\nSystem Boot Information\n	Status: No errors detected\n\nHandle 0x008F, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 8F 00 82 00 FE FF E4 14 5F 16 3C 10 E8 22\n		02 00 FE FF 00 00 04 01 01 01 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x0)/Pci(0x1C,0x0)/Pci(0x0,0x0)\n		NIC.LOM.1.1\n		Network Controller\n		Embedded LOM 1\n\nHandle 0x0090, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 90 00 83 00 FE FF E4 14 5F 16 3C 10 E8 22\n		02 00 FE FF 00 00 04 01 01 02 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x0)/Pci(0x1C,0x0)/Pci(0x0,0x1)\n		NIC.LOM.1.2\n		Network Controller\n		Embedded LOM 1\n\nHandle 0x0091, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 91 00 84 00 FE FF E4 14 D8 16 90 15 12 02\n		02 00 FE FF 00 00 03 01 01 01 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x2)/Pci(0x0,0x0)/Pci(0x0,0x0)\n		NIC.FlexLOM.1.1\n		Network Controller\n		Embedded FlexibleLOM 1\n\nHandle 0x0092, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 92 00 85 00 FE FF E4 14 D8 16 90 15 12 02\n		02 00 FE FF 00 00 03 01 01 02 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x2)/Pci(0x0,0x0)/Pci(0x0,0x1)\n		NIC.FlexLOM.1.2\n		Network Controller\n		Embedded FlexibleLOM 1\n\nHandle 0x0093, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 93 00 86 00 FE FF 2B 10 38 05 90 15 E4 00\n		03 00 FE FF 00 00 09 01 01 01 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x0)/Pci(0x1C,0x4)/Pci(0x0,0x1)\n		PCI.Emb.1.1\n		Embedded Video Controller\n		Embedded Video Controller\n\nHandle 0x0094, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 94 00 87 00 FE FF 05 90 8F 02 3C 10 50 06\n		01 07 FE FF 00 00 07 0A 02 01 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x1)/Pci(0x0,0x0)/Pci(0x0,0x0)\n		RAID.Slot.2.1\n		HPE Smart Array E208i-p SR Gen10\n		Slot 2\n\nHandle 0x0095, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 95 00 88 00 FE FF FF FF FF FF FF FF FF FF\n		FF FF FE FF 00 00 09 0A 01 01 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x1)/Pci(0x2,0x0)/Pci(0x0,0x0)\n		PCI.Slot.1.1\n		Empty slot 1\n		Slot 1\n\nHandle 0x0096, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 96 00 89 00 FE FF FF FF FF FF FF FF FF FF\n		FF FF FE FF 00 00 10 0F 01 01 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x2)/Pci(0x2,0x0)/Pci(0x0,0x0)\n		NVMe.DriveBay.1.1\n		Empty Drive Bay 7\n		NVMe Drive Port 25A Box 1 Bay 1\n\nHandle 0x0097, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 97 00 8A 00 FE FF FF FF FF FF FF FF FF FF\n		FF FF FE FF 00 00 10 0F 01 02 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x2)/Pci(0x3,0x0)/Pci(0x0,0x0)\n		NVMe.DriveBay.1.2\n		Empty Drive Bay 8\n		NVMe Drive Port 25A Box 1 Bay 2\n\nHandle 0x0098, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 98 00 8B 00 FE FF FF FF FF FF FF FF FF FF\n		FF FF FE FF 00 00 10 0F FF 0B FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x3)/Pci(0x2,0x0)/Pci(0x0,0x0)\n		NVMe.DriveBay.255.11\n		Empty Drive Bay 11\n		NVMe Drive 11\n\nHandle 0x0099, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 99 00 8C 00 FE FF FF FF FF FF FF FF FF FF\n		FF FF FE FF 00 00 10 0F FF 0C FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x3)/Pci(0x3,0x0)/Pci(0x0,0x0)\n		NVMe.DriveBay.255.12\n		Empty Drive Bay 12\n		NVMe Drive 12\n\nHandle 0x009A, DMI type 203, 34 bytes\nOEM-specific Type\n	Header and Data:\n		CB 22 9A 00 8D 00 FE FF FF FF FF FF FF FF FF FF\n		FF FF FE FF 00 00 09 0A 03 01 FF FF 01 02 03 04\n		FE FF\n	Strings:\n		PciRoot(0x8)/Pci(0x0,0x0)/Pci(0x0,0x0)\n		PCI.Slot.3.1\n		Empty slot 3\n		Slot 3\n\nHandle 0x009B, DMI type 234, 16 bytes\nOEM-specific Type\n	Header and Data:\n		EA 10 9B 00 FE FF 00 00 01 A0 00 00 00 00 00 00\n\nHandle 0x009C, DMI type 234, 8 bytes\nOEM-specific Type\n	Header and Data:\n		EA 08 9C 00 FE FF 80 05\n\nHandle 0x009D, DMI type 234, 8 bytes\nOEM-specific Type\n	Header and Data:\n		EA 08 9D 00 FE FF 82 05\n\nHandle 0x009E, DMI type 238, 15 bytes\nOEM-specific Type\n	Header and Data:\n		EE 0F 9E 00 04 00 00 A0 01 00 00 02 00 02 01\n	Strings:\n		PciRoot(0x0)/Pci(0x14,0x0)/USB(0x5,0x0)\n\nHandle 0x009F, DMI type 238, 15 bytes\nOEM-specific Type\n	Header and Data:\n		EE 0F 9F 00 05 00 00 A0 01 00 00 02 00 02 01\n	Strings:\n		PciRoot(0x0)/Pci(0x14,0x0)/USB(0x5,0x0)\n\nHandle 0x00A0, DMI type 238, 15 bytes\nOEM-specific Type\n	Header and Data:\n		EE 0F A0 00 06 00 00 A0 03 00 00 01 00 02 01\n	Strings:\n		PciRoot(0x0)/Pci(0x14,0x0)/USB(0x6,0x0)\n\nHandle 0x00A1, DMI type 238, 15 bytes\nOEM-specific Type\n	Header and Data:\n		EE 0F A1 00 06 00 00 A0 03 00 00 01 00 03 01\n	Strings:\n		PciRoot(0x0)/Pci(0x14,0x0)/USB(0x13,0x0)\n\nHandle 0x00A2, DMI type 239, 23 bytes\nOEM-specific Type\n	Header and Data:\n		EF 17 A2 00 A1 00 DA 0B 00 00 08 06 50 29 03 00\n		00 00 00 01 02 03 04\n	Strings:\n		PciRoot(0x0)/Pci(0x14,0x0)/USB(0x13,0x0)\n		HD.SD.1.1\n		Generic USB3.0-CRW\n		Internal SD Card 1\n\nHandle 0x00A3, DMI type 196, 15 bytes\nOEM-specific Type\n	Header and Data:\n		C4 0F A3 00 00 00 00 00 00 00 01 02 00 01 02\n\nHandle 0x00A4, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 A4 00 19 00 FF 01 01 01 00 00 00 02 06 0A\n		4A 00 00 00 00 00 00 00 00\n\nHandle 0x00A5, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 A5 00 1A 00 FF 02 01 02 00 00 00 02 05 08\n		48 00 00 00 00 00 00 00 00\n\nHandle 0x00A6, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 A6 00 1B 00 FF 03 01 03 00 00 00 02 04 06\n		46 00 CE 00 00 00 00 00 00\n\nHandle 0x00A7, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 A7 00 1C 00 FF 04 01 04 00 00 00 02 04 07\n		47 00 00 00 00 00 00 00 00\n\nHandle 0x00A8, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 A8 00 1D 00 FF 05 01 05 00 00 00 01 01 01\n		41 00 00 00 00 00 00 00 00\n\nHandle 0x00A9, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 A9 00 1E 00 FF 06 01 06 00 00 00 01 01 00\n		40 00 00 00 00 00 00 00 00\n\nHandle 0x00AA, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 AA 00 1F 00 FF 07 01 07 00 00 00 01 02 02\n		42 00 00 00 00 00 00 00 00\n\nHandle 0x00AB, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 AB 00 20 00 FF 08 01 08 00 00 00 01 03 04\n		44 00 00 00 00 00 00 00 00\n\nHandle 0x00AC, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 AC 00 21 00 FF 01 02 09 00 00 00 04 06 16\n		56 00 00 00 00 00 00 00 00\n\nHandle 0x00AD, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 AD 00 22 00 FF 02 02 0A 00 00 00 04 05 14\n		54 00 00 00 00 00 00 00 00\n\nHandle 0x00AE, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 AE 00 23 00 FF 03 02 0B 00 00 00 04 04 12\n		52 00 CE 00 00 00 00 00 00\n\nHandle 0x00AF, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 AF 00 24 00 FF 04 02 0C 00 00 00 04 04 13\n		53 00 00 00 00 00 00 00 00\n\nHandle 0x00B0, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 B0 00 25 00 FF 05 02 0D 00 00 00 03 01 0D\n		4D 00 00 00 00 00 00 00 00\n\nHandle 0x00B1, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 B1 00 26 00 FF 06 02 0E 00 00 00 03 01 0C\n		4C 00 00 00 00 00 00 00 00\n\nHandle 0x00B2, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 B2 00 27 00 FF 07 02 0F 00 00 00 03 02 0E\n		4E 00 00 00 00 00 00 00 00\n\nHandle 0x00B3, DMI type 202, 25 bytes\nOEM-specific Type\n	Header and Data:\n		CA 19 B3 00 28 00 FF 08 02 10 00 00 00 03 03 10\n		50 00 00 00 00 00 00 00 00\n\nHandle 0x00B4, DMI type 240, 39 bytes\nOEM-specific Type\n	Header and Data:\n		F0 27 B4 00 8F 00 41 00 12 20 01 00 00 08 00 00\n		00 00 00 03 00 00 00 00 00 00 00 03 00 00 00 00\n		00 00 00 03 00 00 00\n	Strings:\n		20.12.41\n\nHandle 0x00B5, DMI type 240, 39 bytes\nOEM-specific Type\n	Header and Data:\n		F0 27 B5 00 94 00 31 2E 36 36 01 00 00 80 00 00\n		00 00 00 03 00 00 00 00 00 00 00 03 00 00 00 00\n		00 00 00 00 00 00 00\n	Strings:\n		1.66\n\nHandle 0x00B6, DMI type 240, 39 bytes\nOEM-specific Type\n	Header and Data:\n		F0 27 B6 00 91 00 00 00 02 21 01 00 00 80 00 00\n		00 00 00 03 00 00 00 00 00 00 00 03 00 00 00 00\n		00 00 00 03 00 00 00\n	Strings:\n		212.0.103001\n\nHandle 0x00B7, DMI type 233, 41 bytes\nOEM-specific Type\n	Header and Data:\n		E9 29 B7 00 00 00 02 00 D4 C9 EF CE 88 40 00 00\n		00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n		00 00 00 00 00 00 00 00 01\n\nHandle 0x00B8, DMI type 233, 41 bytes\nOEM-specific Type\n	Header and Data:\n		E9 29 B8 00 00 00 02 01 D4 C9 EF CE 88 41 00 00\n		00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n		00 00 00 00 00 00 00 00 02\n\nHandle 0x00B9, DMI type 233, 41 bytes\nOEM-specific Type\n	Header and Data:\n		E9 29 B9 00 00 00 37 00 F4 03 43 C2 49 B0 00 00\n		00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n		00 00 00 00 00 00 00 00 01\n\nHandle 0x00BA, DMI type 233, 41 bytes\nOEM-specific Type\n	Header and Data:\n		E9 29 BA 00 00 00 37 01 F4 03 43 C2 49 B8 00 00\n		00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n		00 00 00 00 00 00 00 00 02\n\nHandle 0xFEFF, DMI type 127, 4 bytes\nEnd Of Table\n\n")
	if ok := assert.NoError(t, err); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Aggregator, "dmidecode 3.1"); !ok {
		t.Fatal()
	}
	if ok := assert.Equal(t, dmi.Tooling.Decoder, DecoderVersion); !ok {
		t.Fatal()
	}
}

func BenchmarkParseMultipleSectionsWithLists(b *testing.B) {
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		parseDMI(sample4)
	}
}

func Test_sectionsFromDMI(t *testing.T) {
	type args struct {
		dmi            *DMI
		sectionTypeStr string
	}
	tests := []struct {
		name         string
		args         args
		wantSections []Section
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotSections := sectionsFromDMI(tt.args.dmi, tt.args.sectionTypeStr); !reflect.DeepEqual(gotSections, tt.wantSections) {
				t.Errorf("sectionsFromDMI() = %v, want %v", gotSections, tt.wantSections)
			}
		})
	}
}

func Test_subSectionsFromDMI(t *testing.T) {
	type args struct {
		dmi             *DMI
		sectionTypeStr  string
		subSectionTitle string
	}
	tests := []struct {
		name            string
		args            args
		wantSubSections []SubSection
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotSubSections := subSectionsFromDMI(tt.args.dmi, tt.args.sectionTypeStr, tt.args.subSectionTitle); !reflect.DeepEqual(gotSubSections, tt.wantSubSections) {
				t.Errorf("subSectionsFromDMI() = %v, want %v", gotSubSections, tt.wantSubSections)
			}
		})
	}
}

func TestBenchmarkParseMultipleSectionsWithLists(t *testing.T) {
	type args struct {
		b *testing.B
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			BenchmarkParseMultipleSectionsWithLists(tt.args.b)
		})
	}
}
