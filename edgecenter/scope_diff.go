package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

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

		resolved, ok := resolveScopeName(ctx, m, scope, name)
		if !ok {
			return nil
		}
		newID = resolved
	}

	if newID == oldID {
		return nil
	}

	if !configuredByID {
		if err := d.SetNew(scope.id, newID); err != nil {
			return fmt.Errorf("set %s from resolved %s: %w", scope.id, scope.name, err)
		}
	}

	if err := d.ForceNew(scope.id); err != nil {
		return fmt.Errorf("mark %s change as force new: %w", scope.id, err)
	}

	return nil
}

func resolveScopeName(ctx context.Context, m interface{}, scope scopeFields, name string) (int, bool) {
	if id, err := strconv.Atoi(name); err == nil {
		return id, true
	}

	config, ok := m.(*Config)
	if !ok {
		return 0, false
	}

	client, err := config.NewCloudClient()
	if err != nil {
		log.Printf("[DEBUG] cannot resolve %s %q during plan: %s", scope.name, name, err)
		return 0, false
	}

	id, err := scope.resolve(ctx, client, name)
	if err != nil {
		log.Printf("[DEBUG] cannot resolve %s %q during plan: %s", scope.name, name, err)
		return 0, false
	}

	return id, true
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
