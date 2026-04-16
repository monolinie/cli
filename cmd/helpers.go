package cmd

import (
	"fmt"
	"strings"

	"github.com/monolinie/cli/internal/dokploy"
)

// findProjectByName looks up a Dokploy project by name (case-insensitive).
func findProjectByName(dk *dokploy.Client, name string) (*dokploy.ProjectDetail, error) {
	projects, err := dk.GetProjects()
	if err != nil {
		return nil, err
	}
	for i := range projects {
		if strings.EqualFold(projects[i].Name, name) {
			return &projects[i], nil
		}
	}
	return nil, fmt.Errorf("project %q not found in Dokploy", name)
}

// findAppInProject returns a matching application in a project.
// If appName is empty, returns the first application found.
func findAppInProject(project *dokploy.ProjectDetail, appName string) (*dokploy.Application, error) {
	for i := range project.Environments {
		for j := range project.Environments[i].Applications {
			app := &project.Environments[i].Applications[j]
			if appName == "" || strings.EqualFold(app.AppName, appName) || strings.EqualFold(app.Name, appName) {
				return app, nil
			}
		}
	}
	if appName != "" {
		return nil, fmt.Errorf("application %q not found in project %q", appName, project.Name)
	}
	return nil, fmt.Errorf("no applications found in project %q", project.Name)
}
