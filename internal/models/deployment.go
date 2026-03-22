package models

type Deployments struct {
	ID           string
	RepoURL      string
	BuildCommand string
	OutputDir    string
	Status       string
}
