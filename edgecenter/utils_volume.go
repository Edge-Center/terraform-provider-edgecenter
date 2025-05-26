package edgecenter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// getVolume retrieves a Volume from the edge cloud service.
// It attempts to find the Volume either by its ID or by its name.
func getVolume(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Volume, error) {
	var (
		volume *edgecloudV2.Volume
		err    error
	)

	name := d.Get(NameField).(string)
	volumeID := d.Get(IDField).(string)

	switch {
	case volumeID != "":
		volume, _, err = clientV2.Volumes.Get(ctx, volumeID)
		if err != nil {
			return nil, err
		}
	default:
		volumeOpts := &edgecloudV2.VolumeListOptions{}
		if metadataK, ok := d.GetOk(MetadataKField); ok {
			volumeOpts.MetadataK = metadataK.(string)
		}

		if metadataRaw, ok := d.GetOk(MetadataKVField); ok {
			typedMetadataKV := make(map[string]string, len(metadataRaw.(map[string]interface{})))
			for k, v := range metadataRaw.(map[string]interface{}) {
				typedMetadataKV[k] = v.(string)
			}
			typedMetadataKVJson, err := json.Marshal(typedMetadataKV)
			if err != nil {
				return nil, err
			}
			volumeOpts.MetadataKV = string(typedMetadataKVJson)
		}

		vols, _, err := clientV2.Volumes.List(ctx, volumeOpts)
		if err != nil {
			return nil, err
		}

		foundVolumes := make([]edgecloudV2.Volume, 0, len(vols))

		for _, v := range vols {
			if v.Name == name {
				foundVolumes = append(foundVolumes, v)
			}
		}

		switch {
		case len(foundVolumes) == 0:
			return nil, errors.New("volume does not exist")

		case len(foundVolumes) > 1:
			var message bytes.Buffer
			message.WriteString("Found volumes:\n")

			for _, vol := range foundVolumes {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", vol.ID))
			}

			return nil, fmt.Errorf("multiple volumes found.\n %s.\n Use volume ID instead of name", message.String())
		}

		volume = &foundVolumes[0]
	}

	return volume, nil
}
