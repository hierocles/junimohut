package moddataset

import (
	"fmt"
	"strings"
)

// ParsePageRef splits a dataset page reference like "Nexus:2400".
func ParsePageRef(ref string) (site string, id int, ok bool) {
	ref = strings.TrimSpace(ref)
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil || id <= 0 {
		return "", 0, false
	}
	return parts[0], id, true
}

// FirstNexusRef returns the first Nexus page ref from refs.
func FirstNexusRef(refs []string) (int, bool) {
	for _, ref := range refs {
		site, id, ok := ParsePageRef(ref)
		if ok && site == "Nexus" {
			return id, true
		}
	}
	return 0, false
}

// PreferNexusRef picks the first Nexus ref, else the first ref of any site.
func PreferNexusRef(refs []string) (site string, id int, ok bool) {
	for _, ref := range refs {
		s, i, o := ParsePageRef(ref)
		if o && s == "Nexus" {
			return s, i, true
		}
	}
	for _, ref := range refs {
		if s, i, o := ParsePageRef(ref); o {
			return s, i, true
		}
	}
	return "", 0, false
}
