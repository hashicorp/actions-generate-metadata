package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	b64 "encoding/base64"

	actions "github.com/sethvargo/go-githubactions"
)

const defaultRepositoryOwner string = "hashicorp"
const defaultMetadataFileName string = "metadata.json"

type input struct {
	branch           string
	filePath         string
	metadataFileName string
	product          string
	releaseMetadata  string
	repo             string
	org              string
	securityScan     string
	sha              string
	version          string
}

type Metadata struct {
	Branch          string `json:"branch"`
	BuildWorkflowId string `json:"buildworkflowid"`
	Product         string `json:"product"`
	Repo            string `json:"repo""`
	Org             string `json:"org"`
	ReleaseMetadata string `json:"releaseMetadata"`
	Revision        string `json:"sha"`
	SecurityScan    string `json:"securityScan"`
	Version         string `json:"version"`
}

func main() {
	in := input{
		branch:           actions.GetInput("branch"),
		filePath:         actions.GetInput("filePath"),
		metadataFileName: actions.GetInput("metadataFileName"),
		product:          actions.GetInput("product"),
		repo:             actions.GetInput("repository"),
		org:              actions.GetInput("repositoryOwner"),
		releaseMetadata:  importFromFile(".release/security-scan.hcl"),
		sha:              actions.GetInput("sha"),
		securityScan:     importFromFile(".release/security-scan.hcl"),
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

	securityScan := in.securityScan
	if securityScan == "" {
		actions.Warningf("Missing security scan configuration.")
	}

	releaseMetadata := in.releaseMetadata
	if securityScan == "" {
		actions.Warningf("Missing release metadata configuration.")
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
		ReleaseMetadata: releaseMetadata,
		Repo:            repository,
		SecurityScan:    securityScan}
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

// importFromFile reads the inputted file from filePath and returns
// it b64encoded.
func importFromFile(filePath string) string {
	scanfile, err := ioutil.ReadFile(filePath)
	if err != nil {
		actions.Fatalf("Failure to read metadata from file:", err)
	}
	return (b64.StdEncoding.EncodeToString(scanfile))
}
