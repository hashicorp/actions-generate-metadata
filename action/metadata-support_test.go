// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
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
