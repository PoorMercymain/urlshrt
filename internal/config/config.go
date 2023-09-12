package config

type Config struct {
	HTTPAddr  AddrWithCheck
	ShortAddr AddrWithCheck
	JSONFile  string
	DSN       string
}

type AddrWithCheck struct {
	Addr   string
	WasSet bool
}

func (a *AddrWithCheck) Set(s string) error {
	a.WasSet = true
	a.Addr = s
	return nil
}

func (a *AddrWithCheck) String() string {
	return a.Addr
}
