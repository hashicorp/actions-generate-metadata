// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractProductName(t *testing.T) {
	cases := []struct {
		name     string
		expected string
	}{
		// consul test cases, verifies consistency for product without variants, including
		// edge cases for rpms and docker artifact names.
		{ // consul dev deb
			name:     "consul_1.13.0~dev-1_arm64.deb",
			expected: "consul_1.13.0-dev",
		},
		{ // consul release deb
			name:     "consul_1.11.3_arm64.deb",
			expected: "consul_1.11.3",
		},
		{ // consul dev rpm
			name:     "consul-1.13.0~dev-1.aarch64.rpm",
			expected: "consul_1.13.0-dev",
		},
		{ // consul release rpm
			name:     "consul-1.11.3.x86_64.rpm",
			expected: "consul_1.11.3",
		},
		{ // consul dev docker
			name:     "consul_default_linux_amd64_1.13.0-dev_77afe0e76e03f6f88376a99936945c0a70e544ac.docker.dev.tar",
			expected: "consul_1.13.0-dev",
		},
		{ // consul release docker
			name:     "consul_default_linux_amd64_1.11.3_36e73cdb6550d4e2cea7548e90ac2b531181ff9d.docker.tar",
			expected: "consul_1.11.3",
		},
		{ // verify the "-" in consul-enterprise is working as expected
			name:     "consul-enterprise_default_linux_386_1.13.0-dev+ent_4700797934aaf631edfeeb58ede73e6484778492.docker.dev.tar",
			expected: "consul-enterprise_1.13.0-dev+ent",
		},

		// consul k8s test cases, verifies that control-plane correctly slots into its own variant
		{ // consul k8s
			name:     "consul-k8s_0.46.0_windows_amd64.zip",
			expected: "consul-k8s_0.46.0",
		},
		{ // consul k8s control plane
			name:     "consul-k8s-control-plane_0.46.0_darwin_arm64.zip",
			expected: "consul-k8s-control-plane_0.46.0",
		},
		{ // consul k8s control plane docker
			name:     "consul-k8s_ubi_linux_amd64_0.46.0_45901d13d0fddf9067ebd1cfb18854c1ef943943.docker.dev.tar",
			expected: "consul-k8s-control-plane_0.46.0",
		},

		// vault test cases, checking both OSS and all of the possible variants of
		// vault enterprise (hsm, fips, hsm.fips, etc)
		{ // regular ol vault dev with some rpm edge cases
			name:     "vault-1.12.0~dev1-1.armv7hl.rpm",
			expected: "vault_1.12.0-dev1",
		},
		{ // vault ent dev
			name:     "vault_1.12.0-dev1+ent_openbsd_arm.zip",
			expected: "vault_1.12.0-dev1+ent",
		},
		{ // vault ent dev fips
			name:     "vault_1.12.0-dev1+ent.fips1402_linux_amd64.zip",
			expected: "vault_1.12.0-dev1+ent.fips1402",
		},
		{ // vault ent dev hsm
			name:     "vault_1.12.0-dev1+ent.hsm_linux_amd64.zip",
			expected: "vault_1.12.0-dev1+ent.hsm",
		},
		{ // vault ent dev hsm fips
			name:     "vault_1.12.0-dev1+ent.hsm.fips1402_linux_amd64.zip",
			expected: "vault_1.12.0-dev1+ent.hsm.fips1402",
		},
		{ // vault-enterprise dev
			name:     "vault-enterprise-1.12.0~dev1+ent-1.armv7hl.rpm",
			expected: "vault-enterprise_1.12.0-dev1+ent",
		},
		{ // vault-enterprise dev hsm
			name:     "vault-enterprise-hsm-1.12.0~dev1+ent-1.x86_64.rpm",
			expected: "vault-enterprise-hsm_1.12.0-dev1+ent",
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.expected, extractProductName(c.name))
	}

}

func TestReleaseSubDirOmitempty(t *testing.T) {
	// When releaseSubDir is set, the JSON output must contain the key.
	withSubDir := &Metadata{
		Product:       "alpha-plugin",
		ReleaseSubDir: "alpha-plugin",
	}
	out, err := json.Marshal(withSubDir)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(out), `"release_sub_dir"`),
		"expected release_sub_dir key in JSON when ReleaseSubDir is set")

	// When releaseSubDir is empty, the JSON output must NOT contain the key (omitempty).
	withoutSubDir := &Metadata{
		Product: "consul",
	}
	out, err = json.Marshal(withoutSubDir)
	assert.NoError(t, err)
	assert.False(t, strings.Contains(string(out), `"release_sub_dir"`),
		"expected no release_sub_dir key in JSON when ReleaseSubDir is empty")
}

func TestSecurityScanAutoDerivesFromReleaseSubDir(t *testing.T) {
	const defaultSecurityScanPath = ".release/security-scan.hcl"

	resolveSecurityScanPath := func(releaseSubDir string) string {
		if releaseSubDir != "" {
			return ".release/" + releaseSubDir + "/security-scan.hcl"
		}
		return defaultSecurityScanPath
	}

	assert.Equal(t, ".release/security-scan.hcl", resolveSecurityScanPath(""),
		"no releaseSubDir: should use default path")

	assert.Equal(t, ".release/vault-plugin-auth-okta/security-scan.hcl",
		resolveSecurityScanPath("vault-plugin-auth-okta"),
		"releaseSubDir set: should use sub-product path")
}

func TestResolveReleaseMetadataFilename(t *testing.T) {
	writeCIHCL := func(t *testing.T, dir, subDir, content string) {
		t.Helper()
		p := filepath.Join(dir, ".release", subDir)
		os.MkdirAll(p, 0755)
		os.WriteFile(filepath.Join(p, "ci.hcl"), []byte(content), 0644)
	}
	chdir := func(t *testing.T, dir string) {
		t.Helper()
		orig, _ := os.Getwd()
		t.Cleanup(func() { os.Chdir(orig) })
		os.Chdir(dir)
	}

	t.Run("no releaseSubDir returns default", func(t *testing.T) {
		assert.Equal(t, "release-metadata.hcl", resolveReleaseMetadataFilename(""))
	})

	t.Run("releaseSubDir with no ci.hcl returns default", func(t *testing.T) {
		chdir(t, t.TempDir())
		assert.Equal(t, "release-metadata.hcl", resolveReleaseMetadataFilename("no-such-product"))
	})

	t.Run("ci.hcl with no config field returns default", func(t *testing.T) {
		dir := t.TempDir()
		writeCIHCL(t, dir, "my-plugin", `
event "promote-staging" {
  action "promote-staging" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-staging"
  }
}`)
		chdir(t, dir)
		assert.Equal(t, "release-metadata.hcl", resolveReleaseMetadataFilename("my-plugin"))
	})

	t.Run("ci.hcl with custom config field returns custom filename", func(t *testing.T) {
		dir := t.TempDir()
		writeCIHCL(t, dir, "trex", `
event "promote-staging" {
  action "promote-staging" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-staging"
    config       = "trex-release-metadata.hcl"
  }
}`)
		chdir(t, dir)
		assert.Equal(t, "trex-release-metadata.hcl", resolveReleaseMetadataFilename("trex"))
	})

	t.Run("ci.hcl with extra blocks before action still resolves config", func(t *testing.T) {
		dir := t.TempDir()
		writeCIHCL(t, dir, "trex", `
event "promote-staging" {
  depends = ["trigger-staging"]
  notification {
    on = "always"
  }
  action "promote-staging" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "promote-staging"
    config       = "trex-release-metadata.hcl"
  }
  promotion-events {}
}`)
		chdir(t, dir)
		assert.Equal(t, "trex-release-metadata.hcl", resolveReleaseMetadataFilename("trex"))
	})
}

func TestReleaseMetadataAutoDerivesFromReleaseSubDir(t *testing.T) {
	assert.Equal(t, "release-metadata.hcl", resolveReleaseMetadataFilename(""),
		"no releaseSubDir: should use default")

	dir := t.TempDir()
	subDir := filepath.Join(dir, ".release", "vault-plugin-auth-okta")
	os.MkdirAll(subDir, 0755)
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	os.Chdir(dir)

	assert.Equal(t, "release-metadata.hcl", resolveReleaseMetadataFilename("vault-plugin-auth-okta"),
		"releaseSubDir set but no ci.hcl: should use default")
}
