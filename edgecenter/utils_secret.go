package edgecenter

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// getSecret retrieves a secret from the edge cloud service.
// It attempts to find the secret either by its ID or by its name.
func getSecret(ctx context.Context, clientV2 *edgecloudV2.Client, d *schema.ResourceData) (*edgecloudV2.Secret, error) {
	var (
		secret *edgecloudV2.Secret
		err    error
	)
	name := d.Get(NameField).(string)
	secretID := d.Get(IDField).(string)

	switch {
	case secretID != "":
		secret, _, err = clientV2.Secrets.Get(ctx, secretID)
		if err != nil {
			return nil, err
		}
	default:
		secrets, _, err := clientV2.Secrets.List(ctx)
		if err != nil {
			return nil, err
		}

		var foundSecrets []edgecloudV2.Secret
		for _, st := range secrets {
			if name == st.Name {
				foundSecrets = append(foundSecrets, st)
			}
		}

		switch {
		case len(foundSecrets) == 0:
			return nil, errors.New("secret does not exist")

		case len(foundSecrets) > 1:
			var message bytes.Buffer
			message.WriteString("Found secrets:\n")

			for _, sec := range foundSecrets {
				message.WriteString(fmt.Sprintf("  - ID: %s\n", sec.ID))
			}

			return nil, fmt.Errorf("multiple secrets found.\n %s.\n Use secret ID instead of name", message.String())
		}

		secret = &foundSecrets[0]
	}

	return secret, nil
}
