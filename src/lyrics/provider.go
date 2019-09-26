package lyrics

// All return the array of usable providers
func All() []Provider {
	return []Provider{
		new(GeniusProvider),
		new(OVHProvider),
	}
}

// Provider defines the generic interface on which every lyrics provider
// should be basing its logic
type Provider interface {
	Name() string
	Query(title, artist string) (string, error)
}
