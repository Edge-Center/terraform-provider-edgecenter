package edgecenter

type SnapshotMetadata = map[string]interface{}

type SnapshotReadOnlyMetadata = []map[string]interface{}

var snapshotReadOnlyTags = map[string]struct{}{
	"bootable":    {},
	"task_id":     {},
	"volume_name": {},
	"volume_type": {},
}

// separateMetadata takes a map of string key-value pairs (md) and separates it into two distinct collections
// Parameters:
// - md: A map of string key-value pairs representing metadata to be separated.
//
// Returns:
// - SnapshotMetadata: A map containing metadata that is mutable.
// - SnapshotReadOnlyMetadata: A slice containing metadata that is read-only.
func separateMetadata(md map[string]string) (SnapshotMetadata, SnapshotReadOnlyMetadata) {
	metadata := make(SnapshotMetadata)
	metadataReadOnly := make(SnapshotReadOnlyMetadata, 0, len(md))

	for k, v := range md {
		if _, ok := snapshotReadOnlyTags[k]; ok {
			metadataReadOnly = append(
				metadataReadOnly,
				SnapshotMetadata{
					"key":       k,
					"value":     v,
					"read_only": true,
				})
			continue
		}

		metadata[k] = v
	}

	return metadata, metadataReadOnly
}
