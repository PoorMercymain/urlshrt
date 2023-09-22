package state

var baseShortAddress string

// InitShortAddress is a function to initialize base address of short URLs.
func InitShortAddress(address string) {
	baseShortAddress = address
}

// GetBaseShortAddress is a function to get the value of base addres for short URLs.
func GetBaseShortAddress() string {
	return baseShortAddress
}
