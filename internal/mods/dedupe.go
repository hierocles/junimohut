package mods

// DedupeByID returns mods with duplicate IDs removed, preserving first occurrence order.
func DedupeByID(list []Mod) []Mod {
	seen := map[string]bool{}
	out := make([]Mod, 0, len(list))
	for _, m := range list {
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		out = append(out, m)
	}
	return out
}

// DedupeByUniqueID returns one mod per SMAPI UniqueID, preferring the canonical or newest copy.
func DedupeByUniqueID(list []Mod) []Mod {
	byUID := map[string]Mod{}
	order := make([]string, 0, len(list))
	for _, m := range list {
		uid := m.Manifest.UniqueID
		if uid == "" {
			uid = m.ID
		}
		key := CanonicalUniqueID(uid)
		if prev, ok := byUID[key]; ok {
			byUID[key] = preferDedupeMod(prev, m)
			continue
		}
		byUID[key] = m
		order = append(order, key)
	}
	out := make([]Mod, 0, len(order))
	for _, key := range order {
		out = append(out, byUID[key])
	}
	return out
}

func preferDedupeMod(a, b Mod) Mod {
	canonical := pickCanonicalDuplicateFolder([]string{a.FolderPath, b.FolderPath})
	switch canonical {
	case a.FolderPath:
		if b.FolderPath != canonical {
			return a
		}
	case b.FolderPath:
		return b
	}
	if versionGreater(a.Manifest.Version, b.Manifest.Version) {
		return a
	}
	if versionGreater(b.Manifest.Version, a.Manifest.Version) {
		return b
	}
	if a.LastUpdated > b.LastUpdated {
		return a
	}
	if b.LastUpdated > a.LastUpdated {
		return b
	}
	return a
}
