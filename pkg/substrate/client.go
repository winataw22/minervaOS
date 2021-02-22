package substrate

// Substrate interface
type Substrate interface {
	GetUser(id uint32) (*User, error)
	GetFarm(id uint32) (*Farm, error)
	GetTwin(id uint32) (*Twin, error)
}
