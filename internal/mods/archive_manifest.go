package mods

// ManifestsFromArchive extracts and parses all manifest.json files from a mod archive.
func ManifestsFromArchive(archivePath string) ([]Manifest, error) {
	return extractManifestsFromArchive(archivePath)
}
