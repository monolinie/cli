package cmd

import (
	"fmt"

	"github.com/monolinie/cli/internal/dokploy"
)

// findProjectByName looks up a Dokploy project by name.
func findProjectByName(dk *dokploy.Client, name string) (*dokploy.ProjectDetail, error) {
	projects, err := dk.GetProjects()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if projects[i].Name == name {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %q not found in Dokploy", name)
}

// findAppInProject returns the first application in a project.
func findAppInProject(project *dokploy.ProjectDetail) (*dokploy.Application, error) {
	for i := range project.Environments {
		if len(project.Environments[i].Applications) > 0 {
			return &project.Environments[i].Applications[0], nil
		}
	}
	return nil, fmt.Errorf("no applications found in project %q", project.Name)
}
