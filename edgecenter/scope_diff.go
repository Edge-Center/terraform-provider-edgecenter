package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

type scopeFields struct {
	id      string
	name    string
	kind    string
	resolve func(context.Context, *edgecloudV2.Client, string) (int, error)
}

var cloudScopes = []scopeFields{
	{id: ProjectIDField, name: ProjectNameField, kind: "project", resolve: resolveProjectIDByName},
	{id: RegionIDField, name: RegionNameField, kind: "region", resolve: resolveRegionIDByName},
}

func RejectScopeChange(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	if d.Id() == "" {
		return nil
	}

	for _, scope := range cloudScopes {
		if err := rejectScopeChange(ctx, d, m, scope); err != nil {
			return err
		}
	}

	return nil
}

func rejectScopeChange(ctx context.Context, d *schema.ResourceDiff, m interface{}, scope scopeFields) error {
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

	return scopeMoveError(scope, oldID, newID)
}

func RequireSameScope(update schema.UpdateContextFunc) schema.UpdateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		for _, scope := range cloudScopes {
			if err := requireSameScope(ctx, d, m, scope); err != nil {
				d.Partial(true)
				return diag.FromErr(err)
			}
		}

		return update(ctx, d, m)
	}
}

func requireSameScope(ctx context.Context, d *schema.ResourceData, m interface{}, scope scopeFields) error {
	oldRaw, _ := d.GetChange(scope.id)
	oldID := oldRaw.(int)
	name := d.Get(scope.name).(string)
	if oldID == 0 || name == "" || !d.HasChange(scope.name) {
		return nil
	}

	newID, err := lookupScopeID(ctx, m, scope, name)
	if err != nil {
		return fmt.Errorf("cannot resolve %s %q: %w", scope.name, name, err)
	}

	if newID != oldID {
		return scopeMoveError(scope, oldID, newID)
	}

	return nil
}

func scopeMoveError(scope scopeFields, oldID, newID int) error {
	return fmt.Errorf(
		"cannot move from %s %d to %s %d: an existing resource cannot change its project or region; revert the change or create a new resource in the target %s",
		scope.kind, oldID, scope.kind, newID, scope.kind,
	)
}

func resolveScopeName(ctx context.Context, m interface{}, scope scopeFields, name string) (int, bool) {
	id, err := lookupScopeID(ctx, m, scope, name)
	if err != nil {
		log.Printf("[DEBUG] cannot resolve %s %q during plan: %s", scope.name, name, err)
		return 0, false
	}

	return id, true
}

func lookupScopeID(ctx context.Context, m interface{}, scope scopeFields, name string) (int, error) {
	if id, err := strconv.Atoi(name); err == nil {
		return id, nil
	}

	config, ok := m.(*Config)
	if !ok {
		return 0, fmt.Errorf("unexpected provider meta %T", m)
	}

	client, err := config.NewCloudClient()
	if err != nil {
		return 0, err
	}

	return scope.resolve(ctx, client, name)
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
