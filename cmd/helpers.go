package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/monolinie/cli/internal/dokploy"
)

var validProjectName = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// validateProjectName checks that a project name is safe to use in DNS, GitHub, paths, etc.
func validateProjectName(name string) error {
	if len(name) == 0 || len(name) > 63 {
		return fmt.Errorf("project name must be 1-63 characters")
	}
	if !validProjectName.MatchString(name) {
		return fmt.Errorf("project name %q contains invalid characters (use letters, numbers, hyphens)", name)
	}
	return nil
}

// findProjectByName looks up a Dokploy project by name (case-insensitive).
func findProjectByName(dk *dokploy.Client, name string) (*dokploy.ProjectDetail, error) {
	if err := validateProjectName(name); err != nil {
		return nil, err
	}
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
