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

// DedupeByUniqueID returns one mod per SMAPI UniqueID, preserving first occurrence order.
func DedupeByUniqueID(list []Mod) []Mod {
	seen := map[string]bool{}
	out := make([]Mod, 0, len(list))
	for _, m := range list {
		uid := m.Manifest.UniqueID
		if uid == "" {
			uid = m.ID
		}
		if seen[uid] {
			continue
		}
		seen[uid] = true
		out = append(out, m)
	}
	return out
}
