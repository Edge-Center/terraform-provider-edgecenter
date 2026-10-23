package edgecenter

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

type scopeFields struct {
	id      string
	name    string
	resolve func(context.Context, *edgecloudV2.Client, string) (int, error)
}

var cloudScopes = []scopeFields{
	{id: ProjectIDField, name: ProjectNameField, resolve: resolveProjectIDByName},
	{id: RegionIDField, name: RegionNameField, resolve: resolveRegionIDByName},
}

func ForceNewOnScopeChange(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	if d.Id() == "" {
		return nil
	}

	for _, scope := range cloudScopes {
		if err := forceNewOnScopeChange(ctx, d, m, scope); err != nil {
			return err
		}
	}

	return nil
}

func forceNewOnScopeChange(ctx context.Context, d *schema.ResourceDiff, m interface{}, scope scopeFields) error {
	oldRaw, newRaw := d.GetChange(scope.id)
	oldID, newID := oldRaw.(int), newRaw.(int)
	if oldID == 0 || !d.NewValueKnown(scope.id) {
		return nil
	}

	configuredByID := d.HasChange(scope.id) && newID != 0
	if !configuredByID {
		if !d.HasChange(scope.name) || !d.NewValueKnown(scope.name) {
			return nil
		}

		name := d.Get(scope.name).(string)
		if name == "" {
			return nil
		}

		config, ok := m.(*Config)
		if !ok {
			return fmt.Errorf("cannot resolve %s %q: provider is not configured", scope.name, name)
		}

		client, err := config.NewCloudClient()
		if err != nil {
			return err
		}

		if newID, err = scope.resolve(ctx, client, name); err != nil {
			return err
		}
	}

	if newID == oldID {
		return nil
	}

	if !configuredByID {
		if err := d.SetNew(scope.id, newID); err != nil {
			return err
		}
	}

	return d.ForceNew(scope.id)
}

func resolveProjectIDByName(ctx context.Context, client *edgecloudV2.Client, name string) (int, error) {
	project, err := GetProjectV2(ctx, client, 0, name)
	if err != nil {
		return 0, err
	}

	return project.ID, nil
}

func resolveRegionIDByName(ctx context.Context, client *edgecloudV2.Client, name string) (int, error) {
	return GetRegionV2(ctx, client, 0, name)
}
