package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v45/github"
	"github.com/sethvargo/go-githubactions"
	"golang.org/x/oauth2"
)

// getArtifacts queries github for artifact list, returning a 2 dimensional slice of
// artifact names paired with their variant type.
// org, repo, and workflowrunID are the github vars passed in to the API for the
// workflow run query, see: https://docs.github.com/en/rest/actions/artifacts#list-workflow-run-artifacts
func getArtifacts(org string, repo string, workflowRunID int64) map[string][]string {

	// Auth to github API.
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		githubactions.Errorf("missing env GITHUB_TOKEN")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient := github.NewClient(tc)

	// Query API for list of artifacts associated with specified build workflow of product.
	opt := &github.ListOptions{PerPage: 100} // 100 is max resposes per page.
	var artifacts []*github.ArtifactList     // each page of results is an element in array.
	for {
		artifactPage, response, err := ghClient.Actions.ListWorkflowRunArtifacts(ctx, org, repo, workflowRunID, opt)
		if err != nil {
			fmt.Println(err)
		}
		artifacts = append(artifacts, artifactPage)
		if response.NextPage == 0 {
			break
		}
		opt.Page = response.NextPage
	}

	// Extract a list of artifact names from the artifact data structure returned from the
	// github API.
	var rawArtifacts []string
	for i := range artifacts {
		for j := range artifacts[i].Artifacts {
			rawArtifacts = append(rawArtifacts, *artifacts[i].Artifacts[j].Name)
		}
	}

	// Parse raw list of artifact names into struct of Artifacts and variant names.
	// For Vault products, associate variant types with their artifacts. All other products
	// do not require variant grouping and thus are associated with variant type "all".
	processedArtifacts := make(map[string][]string)

	// for all other products, we don't need to sort variants, but some builds include
	// metadata files in their artifacts - this removes those from our final list.
	for i := range rawArtifacts {
		switch {
		case strings.Contains(rawArtifacts[i], ".json") || strings.Contains(rawArtifacts[i], ".sig") ||
			strings.Contains(rawArtifacts[i], "_SHA256SUMS"):
			githubactions.Infof("File %v does not match any expected pattern.", rawArtifacts[i])
		default:
			productName := extractProductName(rawArtifacts[i])
			processedArtifacts[productName] = append(processedArtifacts[productName], rawArtifacts[i])
		}
	}

	return processedArtifacts

}

// extractProductName accepts an artifact name, ie. "product_1.2.3-ent_amd64.deb" and
// returns a string containing the name/version of the product, ie. "product_1.2.3-ent"
func extractProductName(rawArtifactName string) string {
	// Extract the product name and relevant extentions with a big ugly regex. Basically,
	// these regexes look for the semver in their input string, then search for a delimater
	// between the variant strings that follow semver (-dev, +ent, +ent.hsm, etc.) and the
	// architecture/file exention. The specifics of this search vary a little for a few special
	// cases. Finally, we return the match minus the arch/file extenion.
	productName := ""
	// RPMs use a . instead of a _ to delinate their extentions.
	if strings.Contains(rawArtifactName, "rpm") {
		re := regexp.MustCompile(`^.*(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*).*?[^\.]*`)
		productName = re.FindString(rawArtifactName)
		productName = correctProductNameRPM(productName)
	} else if strings.Contains(rawArtifactName, "docker") {
		re := regexp.MustCompile(`^.*(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*).*?[^_]*`)
		productName = re.FindString(rawArtifactName)
		productName = correctProductNameDocker(productName)
	} else {
		re := regexp.MustCompile(`^.*(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*).*?[^_]*`)
		productName = re.FindString(rawArtifactName)
	}

	// We need to do some post-processing to make the product name we use in our data more
	// consistent than the product names we have in our artifacts:
	// some files use ~ instead of - in their product text, we replace here as we want product
	// names to stick to the - format.
	productName = strings.ReplaceAll(productName, "~", "-")

	// Some artifact names have a -1 after the semver, we want to remove that for our product
	// name.
	if len(productName) > 1 {
		if productName[len(productName)-2:] == "-1" {
			productName = productName[:len(productName)-2]
		}
	}

	return productName
}

// Docker filenames look something like "product_default_amd_linux_386_1.2.3-dev".
// correctProductNameDocker parses these names and returns a product name which fits
// the norms for our other artifacts, such as "product_1.2.3-dev" for the example.
func correctProductNameDocker(productName string) string {
	// List of strings indicitive of docker cruft - add new strings here if a team
	// changes their docker naming and their artifacts aren't getting caught.
	// Map of structs is a little odd looking, but allows easy lookups as we loop
	// through the terms in productName, preventing an On^2 loop.
	var dockerExceptions = map[string]struct{}{
		"default": {}, "release": {}, "release-default": {}, "ubi": {},
	}
	// Split the original string along the delimeter "_", then loop through 1 word at a time.
	substrings := strings.Split(productName, "_")
	var newProductName string
	for i, word := range substrings {
		// When we find a term that signifies docker filename cruft, loop through the
		// remainder of the list looking for the semver and setting anything else to an
		// empty string. When we find the semver, break out of this loop.
		if _, ok := dockerExceptions[word]; ok {
			re := regexp.MustCompile(`(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*).*?[^_]*`)
			for j := i; j < len(substrings); j++ {
				found := re.MatchString(substrings[j])
				if found {
					break
				} else {
					substrings[j] = ""
				}
			}
		}
		// build a new string from the current substring index, ignoring empty strings and
		// re-inserting the _ which we removed as a delimeter.
		if i == 0 {
			newProductName += substrings[i]
		} else if substrings[i] != "" {
			newProductName = newProductName + "_" + substrings[i]
		}
	}

	return newProductName
}

// rpms use a - instead of an _ as the delimeter before the semver. This changes it to
// "_" for consistency.
func correctProductNameRPM(productName string) string {
	// Solving this required some hard to read logic, documenting heavily for readability:
	// Create a regex that finds a substring from start of string through semver.
	re := regexp.MustCompile(`^.*-(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`)
	// Break the product name into 2 halves with semver as a fulcrum. This produces 2
	// halves that look something like: "product-1.2.3" and "-dev-1" or
	// "product-1.2.3" and "".
	prefixString := strings.Join(re.FindAllString(productName, -1), "")
	postfixString := strings.Join(re.Split(productName, -1), "")
	// Find the last "-" in the prefix substring, verify that it exists, and rebuild the
	// prefix string around that index replacing "-" with "_"
	charIndex := strings.LastIndex(prefixString, "-")
	if charIndex > 0 {
		prefixString = prefixString[:charIndex] + "_" + prefixString[charIndex+1:]
	} else {
		githubactions.Warningf("Attempt to correct RPM product name for consistency failed.")
	}
	// rebuild the original product name using the modified prefix.
	return prefixString + postfixString
}
