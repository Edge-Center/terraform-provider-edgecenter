package edgecenter

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

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

// getSnapshot retrieves a Snapshot from the edge cloud service.
// It attempts to find the Snapshot either by its ID or by its name.
func getSnapshot(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Snapshot, error) {
	var (
		snapshot *edgecloudV2.Snapshot
		err      error
	)

	snapshotID := d.Get(SnapshotIDField).(string)
	volumeID := d.Get(VolumeIDField).(string)
	name := d.Get(NameField).(string)

	switch {
	case snapshotID != "":
		snapshot, _, err = clientV2.Snapshots.Get(ctx, snapshotID)
		if err != nil {
			return nil, err
		}

	default:
		snapshotsOpts := &edgecloudV2.SnapshotListOptions{VolumeID: volumeID}

		allSnapshots, _, err := clientV2.Snapshots.List(ctx, snapshotsOpts)
		if err != nil {
			return nil, err
		}

		foundSnapshots := make([]edgecloudV2.Snapshot, 0, len(allSnapshots))

		for _, s := range allSnapshots {
			if name == s.Name {
				foundSnapshots = append(foundSnapshots, s)
			}
		}

		switch {
		case len(foundSnapshots) == 0:
			return nil, errors.New("snapshot does not exist")

		case len(foundSnapshots) > 1:
			var message bytes.Buffer
			message.WriteString("Found snapshots:\n")

			for _, fSG := range foundSnapshots {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", fSG.ID))
			}

			return nil, fmt.Errorf("multiple snapshots found.\n %s.\n Use snapshot ID instead of name", message.String())
		}

		snapshot = &foundSnapshots[0]
	}

	return snapshot, nil
}
