package lyrics

// Providers is an exported array of usable providers
var Providers = []Provider{
	new(GeniusProvider),
	new(OVHProvider),
}

// Provider defines the generic interface on which every lyrics provider
// should be basing its logic
type Provider interface {
	Name() string
	Query(title, artist string) (string, error)
}
