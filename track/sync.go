package track

// SyncOptions wraps settings for a single track synchronization
type SyncOptions struct {
	Source bool	
	Metadata    bool  
	Normalization bool
}

// SyncOptionsFlush returns a SyncOptions pointer for flushing tracks
func SyncOptionsFlush() *SyncOptions {
	opts := SyncOptionsDefault()
	opts.Source = true
	opts.Metadata = true
	return opts
}

// SyncOptionsDefault returns the default SyncOptions pointer
func SyncOptionsDefault() *SyncOptions {
	return &SyncOptions{Source: false, Metadata: false, Normalization: true}
}
