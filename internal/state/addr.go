package state

var baseShortAddress string

func InitShortAddress(address string) {
	baseShortAddress = address
}

func GetBaseShortAddress() string {
	return baseShortAddress
}
