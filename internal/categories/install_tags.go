package categories

const (
	TagFashionSense = "tag-fashion-sense"
	TagItems        = "tag-items"
)

// MergeInstallSuggestedTags combines Nexus category tags with manifest-based overrides.
// When fashionSense is true and the Fashion Sense tag exists, tag-items is removed.
func MergeInstallSuggestedTags(nexusTagIDs []string, fashionSense bool, knownTags map[string]bool) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(nexusTagIDs)+1)
	for _, tagID := range nexusTagIDs {
		if tagID == "" || seen[tagID] || !knownTags[tagID] {
			continue
		}
		if fashionSense && tagID == TagItems {
			continue
		}
		seen[tagID] = true
		out = append(out, tagID)
	}
	if fashionSense && knownTags[TagFashionSense] && !seen[TagFashionSense] {
		out = append(out, TagFashionSense)
	}
	return out
}
