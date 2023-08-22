// config package contains some types for the app configuration.
package config

// Config type contains some of the app's configuration info.
type Config struct {
	HTTPAddr  AddrWithCheck
	ShortAddr AddrWithCheck
	JSONFile  string
	DSN       string
}

// AddrWithCheck is a type which represents address and adiitional variable to check if the address was set.
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
