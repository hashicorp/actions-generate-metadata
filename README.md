# generate-metadata action

This action creates JSON file containing metadata information.

## Example of the generated metadata.json file
```json
{
"repository": "consul-terraform-sync",
"repositoryOwner": "hashicorp",
"sha": "4671a6594f2a2650f066489a4fbfe35c3b1e3d35",
"version": "1.0.3",
"buildWorkflowId": "1284662138"
"product": "consul",
"branch": "main",

}
```

## Usage

See [action.yaml](https://github.com/hashicorp/actions-generate-metadata/blob/main/action.yml)

### Basic usage example

```yaml
- name: Generate metadata file
  uses: hashicorp/actions-generate-metadata@main
  id: execute
  with:
    repository: consul-terraform-sync
    version: 1.2.3
```

### Usage example to create metadata.json file using command in the version input
```yaml
- name: Generate metadata file
  uses: hashicorp/actions-generate-metadata@main
  id: execute
  with:
    repository: consul-terraform-sync
    version: make version
    metadataFileName: metadata.json
```

## Inputs

* **`repository`** - (required). The repository name for collecting the metadata
* **`version`** - (required). Indicates the version to be set in the metadata file. Can also accept the command which will set the version (e.g "make version")*

* **`branch`** - (optional). Github Branch of changes
* **`filePath`** - (optional). Existing path that denotes the location of the metadata file to be created. The action will not create specified directory if it not exist. Default is set to Github action root path.
* **`product`** - (optional). The product binary name
* **`repositoryOwner`** - (optional). The repository owner (organization or user). Default is set to "hashicorp" organization
* **`metadataFileName`** - (optional). The name of the file produced by the action. The generated file will have a JSON format. Default is set to "metadata.json"

*Example command for the `version` input is provided [here](https://github.com/hashicorp/actions-generate-metadata#create-metadatajson-file-using-command-in-the-version-input)


## Outputs
* `filepath` - The path where `metadataFileName` was created after action finished executing.
