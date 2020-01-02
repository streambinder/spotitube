package system

// MapGroups returns a mapping between regex SubexpNames and match
func MapGroups(match, names []string) map[string]string {
	if len(match) == 0 {
		return make(map[string]string, 0)
	}

	match, names = match[1:], names[1:]
	matchMap := make(map[string]string, len(match))
	for i := range names {
		matchMap[names[i]] = match[i]
	}

	return matchMap
}
