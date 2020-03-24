package client

import (
	"fmt"
	"net/url"
)

// NodeFilter used to build a query for node list
type NodeFilter struct {
	farm    *int64
	country *string
	city    *string
	cru     *int64
	mru     *int64
	sru     *int64
	hru     *int64
	proofs  *bool
}

// WithFarm filter with farm
func (n NodeFilter) WithFarm(id int64) NodeFilter {
	n.farm = &id
	return n
}

//WithCountry filter with country
func (n NodeFilter) WithCountry(country string) NodeFilter {
	n.country = &country
	return n
}

//WithCity filter with city
func (n NodeFilter) WithCity(city string) NodeFilter {
	n.city = &city
	return n
}

//WithCRU filter with CRU
func (n NodeFilter) WithCRU(cru int64) NodeFilter {
	n.cru = &cru
	return n
}

// WithMRU filter with mru
func (n NodeFilter) WithMRU(sru int64) NodeFilter {
	n.sru = &sru
	return n
}

// WithHRU filter with HRU
func (n NodeFilter) WithHRU(hru int64) NodeFilter {
	n.hru = &hru
	return n
}

// WithProofs filter with proofs
func (n NodeFilter) WithProofs(proofs bool) NodeFilter {
	n.proofs = &proofs
	return n
}

// Apply fills query
func (n NodeFilter) Apply(query url.Values) {

	if n.farm != nil {
		query.Set("farm", fmt.Sprint(*n.farm))
	}

	if n.country != nil {
		query.Set("country", *n.country)
	}

	if n.city != nil {
		query.Set("city", *n.city)
	}

	if n.cru != nil {
		query.Set("cru", fmt.Sprint(*n.cru))
	}

	if n.mru != nil {
		query.Set("mru", fmt.Sprint(*n.mru))
	}

	if n.sru != nil {
		query.Set("sru", fmt.Sprint(*n.sru))
	}

	if n.hru != nil {
		query.Set("hru", fmt.Sprint(*n.hru))
	}

	if n.proofs != nil {
		query.Set("proofs", fmt.Sprint(*n.proofs))
	}
}
