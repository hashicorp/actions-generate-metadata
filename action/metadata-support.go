package main

import (
	"context"
	"os"
	"strings"

	"github.com/google/go-github/v45/github"
	"github.com/sethvargo/go-githubactions"
	"golang.org/x/oauth2"
)

// getArtifacts queries github for artifact list, returning a 2 dimensional slice of
// artifact names paired with their variant type.
// org, repo, and workflowrunID are the github vars passed in to the API for the
// workflow run query, see: https://docs.github.com/en/rest/actions/artifacts#list-workflow-run-artifacts
func getArtifacts(org string, repo string, workflowRunID int64) [][]string {

	// Auth to github API.
	token := os.Getenv("CRT_GITHUB_TOKEN")
	if token == "" {
		githubactions.Errorf("missing env CRT_GITHUB_TOKEN")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	// Query API for list of artifacts associated with specified build workflow of product.
	artifacts, _, err := ghClient.Actions.ListWorkflowRunArtifacts(ctx, org, repo, workflowRunID, &github.ListOptions{})
	if err != nil {
		githubactions.Errorf(err)
	}

	// Extract a list of artifact names from the artifact data structure returned from the
	// github API.
	var rawArtifacts []string
	for i := range artifacts.Artifacts {
		rawArtifacts = append(rawArtifacts, *artifacts.Artifacts[i].Name)
	}

	// Parse raw list of artifact names into struct of Artifacts and variant names.
	// For Vault products, associate variant types with their artifacts. All other products
	// do not require variant grouping and thus are associated with variant type "all".
	var processedArtifacts [][]string
	if repo == "vault-enterprise" {
		for i := range rawArtifacts {
			switch {
			case strings.Contains(rawArtifacts[i], "ent.hsm.fips"):
				processedArtifacts = append(processedArtifacts, []string{"ent.hsm.fips", rawArtifacts[i]})
			case strings.Contains(rawArtifacts[i], "ent.fips"):
				processedArtifacts = append(processedArtifacts, []string{"ent.fips", rawArtifacts[i]})
			case strings.Contains(rawArtifacts[i], "ent.hsm"):
				processedArtifacts = append(processedArtifacts, []string{"ent.hsm", rawArtifacts[i]})
			case strings.Contains(rawArtifacts[i], "ent"):
				processedArtifacts = append(processedArtifacts, []string{"ent", rawArtifacts[i]})
			default:
				githubactions.Infof("File %v does not match any expected pattern.", rawArtifacts[i])
			}
		}
	} else if repo == "consul-k8s" {
		for i := range rawArtifacts {
			switch {
			case strings.Contains(rawArtifacts[i], "consul-k8s-control-plane"):
				processedArtifacts = append(processedArtifacts, []string{"consul-k8s-control-plane", rawArtifacts[i]})
			case strings.Contains(rawArtifacts[i], "consul-k8s"):
				processedArtifacts = append(processedArtifacts, []string{"consul-k8s", rawArtifacts[i]})
			default:
				githubactions.Infof("File %v does not match any expected pattern.", rawArtifacts[i])
			}
		}
	} else {
		// for all other products, we don't need to sort variants, but some builds include
		// metadata files in their artifacts - this removes those from our final list.
		for i := range rawArtifacts {
			switch {
			case strings.Contains(rawArtifacts[i], ".json") || strings.Contains(rawArtifacts[i], ".sig") ||
				strings.Contains(rawArtifacts[i], "_SHA256SUMS"):
				githubactions.Infof("File %v does not match any expected pattern.", rawArtifacts[i])
			default:
				processedArtifacts = append(processedArtifacts, []string{"all", rawArtifacts[i]})
			}
		}
	}

	return processedArtifacts
}
