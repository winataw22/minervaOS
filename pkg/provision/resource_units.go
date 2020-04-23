package provision

// ResourceUnits type
type ResourceUnits string

// ResourcesUnits are the units used to compute how much
// capacity is reserved on the system
var (
	ResourceUnitsCRU = ResourceUnits("CRU")
	ResourceUnitsMRU = ResourceUnits("MRU")
	ResourceUnitsHRU = ResourceUnits("HRU")
	ResourceUnitsSRU = ResourceUnits("SRU")
)
