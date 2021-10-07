package main

import (
	"bytes"
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

type input struct {
	branch           string
	filePath         string
	metadataFileName string
	product          string
	repository       string
	org              string
	sha              string
	version          string
}

type Metadata struct {
	Branch          string `json:"branch"`
	BuildWorkflowId string `json:"buildWorkflowId"`
	Product         string `json:"product"`
	Repository      string `json:"repository""`
	Org             string `json:"org"`
	Revision        string `json:"sha"`
	Version         string `json:"version"`
}

func main() {
	in := input{
		branch:           actions.GetInput("branch"),
		filePath:         actions.GetInput("filePath"),
		metadataFileName: actions.GetInput("metadataFileName"),
		product:          actions.GetInput("product"),
		repository:       actions.GetInput("repository"),
		org:              actions.GetInput("org"),
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

func createMetadataJson(in input) string {
	branch := in.branch
	if branch == "" && os.Getenv("GITHUB_HEAD_REF") == "" {
		branch = "main"
	} else {
		branch = os.Getenv("GITHUB_REF")
	}
	actions.Infof("Working branch %v\n", branch)

	file := in.metadataFileName
	if file == "" {
		file = defaultMetadataFileName
	}
	filePath := path.Join(in.filePath, file)
	product := in.product
	if product == "" {
		actions.Warningf("Missing input 'product'")
	}
	sha := in.sha
	if sha == "" {
		sha = os.Getenv("GITHUB_SHA")
	}
	actions.Infof("Working sha %v\n", sha)

	repository := in.repository
	if repository == "" {
		sha = os.Getenv("GITHUB_REPOSITORY")
	}

	org := in.org
	if org == "" {
		org = defaultRepositoryOwner
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

	m := &Metadata{
		Product:         product,
		Org:             org,
		Revision:        sha,
		BuildWorkflowId: runId,
		Version:         version,
		Branch:          branch,
		Repository:      repository}
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
