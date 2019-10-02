package directory

// TfgridFarmer1 jsx schema
type TfgridFarmer1 struct {
	OwnerThreebotID string   `json:"owner_threebot_id"`
	IyoOrganization string   `json:"iyo_organization"`
	Name            string   `json:"name"`
	WalletAddresses []string `json:"wallet_addresses"`
	Email           string   `json:"email"`
}
