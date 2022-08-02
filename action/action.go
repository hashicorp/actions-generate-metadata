package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	actions "github.com/sethvargo/go-githubactions"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

const defaultRepositoryOwner string = "hashicorp"
const defaultMetadataFileName string = "metadata.json"
const releasePath = ".release"

type input struct {
	branch           string
	filePath         string
	metadataFileName string
	product          string
	repo             string
	org              string
	sha              string
	version          string
}

type Metadata struct {
	Branch          string            `json:"branch"`
	BuildWorkflowId string            `json:"buildworkflowid"`
	Product         string            `json:"product"`
	Repo            string            `json:"repo"`
	Org             string            `json:"org"`
	ReleaseMetadata map[string]string `json:"release-metadata"`
	Revision        string            `json:"sha"`
	SecurityScan    map[string]string `json:"security-scan"`
	Version         string            `json:"version"`
}

func main() {
	in := input{
		branch:           actions.GetInput("branch"),
		filePath:         actions.GetInput("filePath"),
		metadataFileName: actions.GetInput("metadataFileName"),
		product:          actions.GetInput("product"),
		repo:             actions.GetInput("repository"),
		org:              actions.GetInput("repositoryOwner"),
		sha:              actions.GetInput("sha"),
		version:          actions.GetInput("version"),
	}
	generatedFile := createMetadataJson(in)

	if checkFileIsExist(generatedFile) {
		actions.SetOutput("filepath", generatedFile)
		actions.SetEnv("filepath", generatedFile)
		actions.Infof("Successfully created %v file\n", generatedFile)
	} else {
		actions.Fatalf("File %v does not exist", generatedFile)
	}
}

func checkFileIsExist(filepath string) bool {
	fileInfo, err := os.Stat(filepath)

	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		actions.Fatalf("failed to read file: %v", filepath)
	}
	// Return false if the fileInfo says the file path is a directory
	return !fileInfo.IsDir()
}

func b64EncodeReleaseMetadata(productVersions []string) (map[string]string, error) {
	const (
		defaultReleaseMetadata = releasePath + "/release-metadata.hcl"
	)

	b64, err := b64EncodeFile(defaultReleaseMetadata)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]string)
	for _, version := range productVersions {
		metadata[version] = b64
	}

	return metadata, nil
}

func b64EncodeSecurityScan(productVersions []string) (map[string]string, error) {
	const (
		defaultSecurityScan = releasePath + "/security-scan.hcl"
	)

	b64, err := b64EncodeFile(defaultSecurityScan)
	if err != nil {
		return nil, err
	}

	scan := make(map[string]string)
	for _, version := range productVersions {
		scan[version] = b64
	}

	return scan, nil
}

func b64EncodeFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(data)

	return b64, nil
}

func createMetadataJson(in input) string {
	branch := in.branch
	actions.Infof("GITHUB_HEAD_REF %v\n", os.Getenv("GITHUB_HEAD_REF"))
	actions.Infof("GITHUB_REF %v\n", os.Getenv("GITHUB_REF"))
	if branch == "" && os.Getenv("GITHUB_HEAD_REF") == "" {
		branch = strings.TrimPrefix(os.Getenv("GITHUB_REF"), "refs/heads/")
	} else {
		branch = os.Getenv("GITHUB_HEAD_REF")
	}

	actions.Infof("Working branch %v\n", branch)

	file := in.metadataFileName
	if file == "" {
		file = defaultMetadataFileName
	}
	filePath := path.Join(in.filePath, file)

	product := in.product
	if product == "" {
		actions.Fatalf("Missing input 'product' value")
	}
	sha := in.sha
	if sha == "" {
		sha = os.Getenv("GITHUB_SHA")
	}
	actions.Infof("Working sha %v\n", sha)

	org := in.org
	if org == "" {
		org = defaultRepositoryOwner
	}
	repository := in.repo
	if repository == "" {
		repository = strings.Split(os.Getenv("GITHUB_REPOSITORY"), "/")[1]
	}

	runId := os.Getenv("GITHUB_RUN_ID")
	if runId == "" {
		actions.Fatalf("GITHUB_RUN_ID is empty")
	}

	version := in.version
	if version == "" {
		actions.Fatalf("The version or version command is not provided")
	} else if strings.Contains(version, " ") {
		version = getVersion(version)
	}
	actions.Infof("Working version %v\n", version)

	actions.Infof("Creating metadata file in %v\n", filePath)

	// For release-metadata.hcl and security-scan.hcl, we currently only recognize one product version
	// per repository, but plan to support multiple in the future. Until that is supported, assume all
	// use the same metadata and security-scan configuration group it under the base product-version
	productVersion := product + "_" + version

	metadata, err := b64EncodeReleaseMetadata([]string{productVersion})
	if err != nil {
		actions.Fatalf(err.Error())
	}
	scan, err := b64EncodeSecurityScan([]string{productVersion})
	if err != nil {
		actions.Fatalf(err.Error())
	}

	m := &Metadata{
		Product:         product,
		Org:             org,
		Revision:        sha,
		BuildWorkflowId: runId,
		Branch:          branch,
		Version:         version,
		Repo:            repository,
		ReleaseMetadata: metadata,
		SecurityScan:    scan,
	}
	output, err := json.MarshalIndent(m, "", "\t\t")

	if err != nil {
		actions.Fatalf("JSON marshal failure. Error:%v\n", output, err)
	} else {
		err = ioutil.WriteFile(filePath, output, 0644)
		if err != nil {
			actions.Fatalf("Failed writing data into %v file. Error: %v\n", in.metadataFileName, err)
		}
	}
	return filePath
}

func getVersion(command string) string {
	version := execCommand(strings.Fields(command)...)
	if version == "" {
		actions.Fatalf("Failed to setup version using %v command", command)
	}
	return strings.TrimSuffix(version, "\n")
}

func execCommand(args ...string) string {
	name := args[0]
	stderr := new(bytes.Buffer)
	stdout := new(bytes.Buffer)

	cmd := exec.Command(name, args[1:]...)
	cmd.Stderr = stderr
	cmd.Stdout = stdout
	err := cmd.Run()
	actions.Infof("Running %v command: %v\nstdout: %v\nstderr: %v\n", name, cmd,
		strings.TrimSpace(string(stdout.Bytes())), strings.TrimSpace(string(stderr.Bytes())))

	if err != nil {
		actions.Fatalf("Failed to run %v command %v: %v", name, cmd, err)
	}
	return string(stdout.Bytes())
}
