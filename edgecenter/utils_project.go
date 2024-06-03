package edgecenter

import (
	"context"
	"fmt"
	"log"
	"strconv"

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

func GetProjectV2(
	ctx context.Context,
	clientV2 *edgecloudV2.Client,
	projectID,
	projectName string,
) (*edgecloudV2.Project, error) {
	if projectID != "" {
		p, err := GetProjectByIDV2(ctx, clientV2, projectID)
		if err != nil {
			return nil, err
		}
		return p, nil
	}
	allProjects, _, err := clientV2.Projects.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot get project. Error: %s", err.Error())
	}

	var foundProjects []edgecloudV2.Project
	for _, p := range allProjects {
		if projectName == p.Name {
			foundProjects = append(foundProjects, p)
		}
	}

	if len(foundProjects) == 0 {
		return nil, fmt.Errorf("project with name %s does not exist", projectName)
	} else if len(foundProjects) > 1 {
		return nil, fmt.Errorf("multiple project found with name %s. Use id instead of name", projectName)
	}

	return &foundProjects[0], nil
}

// findProjectByNameV2 searches for a project with the specified name and id in the provided project slice.
// Use new version Edgecenterclient-go V2.
// Returns the project if found, otherwise returns an error.
func findProjectByNameV2(
	arr []edgecloudV2.Project,
	name string,
) (*edgecloudV2.Project, error) {
	// TODO remove when upgrading to a new version golang and use slices.IndexFunc - https://tracker.yandex.ru/CLOUDDEV-456.
	index := IndexFunc(arr, func(p edgecloudV2.Project) bool { return p.Name == name })
	if index != -1 {
		return &arr[index], nil
	}

	return nil, fmt.Errorf("project with name %s not found", name)
}

// findProjectByIDV2 searches for a project with the specified name and id in the provided project slice.
// Use new version Edgecenterclient-go V2.
// Returns the project if found, otherwise returns an error.
func findProjectByIDV2(
	arr []edgecloudV2.Project,
	id string,
) (*edgecloudV2.Project, error) {
	// TODO remove when upgrading to a new version golang and use slices.IndexFunc - https://tracker.yandex.ru/CLOUDDEV-456.
	index := IndexFunc(arr, func(p edgecloudV2.Project) bool {
		return strconv.Itoa(p.ID) == id
	})
	if index != -1 {
		return &arr[index], nil
	}

	return nil, fmt.Errorf("project with id %s not found", id)
}

// GetProjectByNameV2 returns a valid project for a resource.
// Use new version Edgecenterclient-go V2.
// If the projectID is provided, it will be returned directly.
// If projectName is provided instead, the function will search for the project by name and return its.
// Returns an error if the project is not found or there is an issue with the client.
func GetProjectByNameV2(
	ctx context.Context,
	client *edgecloudV2.Client,
	projectName string,
) (*edgecloudV2.Project, error) {
	log.Println("[DEBUG] Try to get project ID")
	projectsList, _, err := client.Projects.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Projects: %v", projectsList)

	project, err := findProjectByNameV2(projectsList, projectName)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] The attempt to get the project is successful: projectID=%d", project.ID)

	return project, nil
}

// GetProjectByIDV2 returns a valid project for a resource.
// Use new version Edgecenterclient-go V2.
// If projectName is provided instead, the function will search for the project by id and return its.
// Returns an error if the project is not found or there is an issue with the client.
func GetProjectByIDV2(
	ctx context.Context,
	client *edgecloudV2.Client,
	projectID string,
) (*edgecloudV2.Project, error) {
	log.Println("[DEBUG] Try to get project ID")
	projectsList, _, err := client.Projects.List(ctx, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Projects: %v", projectsList)

	project, err := findProjectByIDV2(projectsList, projectID)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] The attempt to get the project is successful: projectID=%d", project.ID)

	return project, nil
}
