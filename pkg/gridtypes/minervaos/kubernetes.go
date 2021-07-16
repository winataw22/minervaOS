package zos

import (
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

var vmSize = map[uint8]gridtypes.Capacity{
	1: {
		CRU: 1,
		MRU: 2 * gridtypes.Gigabyte,
	},
	2: {
		CRU: 2,
		MRU: 4 * gridtypes.Gigabyte,
	},
	3: {
		CRU: 2,
		MRU: 8 * gridtypes.Gigabyte,
	},
	4: {
		CRU: 2,
		MRU: 5 * gridtypes.Gigabyte,
	},
	5: {
		CRU: 2,
		MRU: 8 * gridtypes.Gigabyte,
	},
	6: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
	},
	7: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
	},
	8: {
		CRU: 4,
		MRU: 16 * gridtypes.Gigabyte,
	},
	9: {
		CRU: 8,
		MRU: 32 * gridtypes.Gigabyte,
	},
	10: {
		CRU: 8,
		MRU: 32 * gridtypes.Gigabyte,
	},
	11: {
		CRU: 8,
		SRU: 800 * gridtypes.Gigabyte,
	},
	12: {
		CRU: 1,
		MRU: 64 * gridtypes.Gigabyte,
	},
	13: {
		CRU: 1,
		MRU: 64 * gridtypes.Gigabyte,
	},
	14: {
		CRU: 1,
		SRU: 800 * gridtypes.Gigabyte,
	},
	15: {
		CRU: 1,
		SRU: 25 * gridtypes.Gigabyte,
	},
	16: {
		CRU: 2,
		SRU: 50 * gridtypes.Gigabyte,
	},
	17: {
		CRU: 4,
		SRU: 50 * gridtypes.Gigabyte,
	},
	18: {
		CRU: 1,
		MRU: 1 * gridtypes.Gigabyte,
	},
}
