package edgecenter

import (
	"context"
	"fmt"
	"log"

	edgecloud "github.com/Edge-Center/edgecentercloud-go"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter"
	"github.com/Edge-Center/edgecentercloud-go/edgecenter/project/v1/projects"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

// findProjectByName searches for a project with the specified name in the provided project slice.
// Returns the project ID if found, otherwise returns an error.
// ToDo Remove after migrate to Edgecenterclient-go V2.
func findProjectByName(arr []projects.Project, name string) (int, error) {
	for _, el := range arr {
		if el.Name == name {
			return el.ID, nil
		}
	}
	return 0, fmt.Errorf("project with name %s not found", name)
}

// GetProject returns a valid project ID for a resource.
// If the projectID is provided, it will be returned directly.
// If projectName is provided instead, the function will search for the project by name and return its ID.
// Returns an error if the project is not found or there is an issue with the client.
// ToDo Remove after migrate to Edgecenterclient-go V2.
func GetProject(provider *edgecloud.ProviderClient, projectID int, projectName string) (int, error) {
	log.Println("[DEBUG] Try to get project ID")
	if projectID != 0 {
		return projectID, nil
	}
	client, err := edgecenter.ClientServiceFromProvider(provider, edgecloud.EndpointOpts{
		Name:    ProjectPoint,
		Region:  0,
		Project: 0,
		Version: VersionPointV1,
	})
	if err != nil {
		return 0, err
	}
	projectsList, err := projects.ListAll(client)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] Projects: %v", projectsList)
	projectID, err = findProjectByName(projectsList, projectName)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] The attempt to get the project is successful: projectID=%d", projectID)

	return projectID, nil
}

// findProjectByNameV2 searches for a project with the specified name in the provided project slice.
// Use new version Edgecenterclient-go V2.
// Returns the project ID if found, otherwise returns an error.
func findProjectByNameV2(arr []edgecloudV2.Project, name string) (int, error) {
	for _, el := range arr {
		if el.Name == name {
			return el.ID, nil
		}
	}
	return 0, fmt.Errorf("project with name %s not found", name)
}

// GetProjectV2 returns a valid project ID for a resource.
// Use new version Edgecenterclient-go V2.
// If the projectID is provided, it will be returned directly.
// If projectName is provided instead, the function will search for the project by name and return its ID.
// Returns an error if the project is not found or there is an issue with the client.
func GetProjectV2(ctx context.Context, client *edgecloudV2.Client, projectID int, projectName string) (int, error) {
	log.Println("[DEBUG] Try to get project ID")
	if projectID != 0 {
		return projectID, nil
	}

	projectsList, _, err := client.Projects.List(ctx, nil)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] Projects: %v", projectsList)
	projectID, err = findProjectByNameV2(projectsList, projectName)
	if err != nil {
		return 0, err
	}
	log.Printf("[DEBUG] The attempt to get the project is successful: projectID=%d", projectID)

	return projectID, nil
}
