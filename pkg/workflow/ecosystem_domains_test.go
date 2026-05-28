//go:build !integration

package workflow

import (
	"slices"
	"testing"
)

type ecosystemDomainExpansionTestCase struct {
	name          string
	allowed       []string
	expected      []string
	excluded      []string
	expectedCount int
	context       string
}

var ecosystemDomainInfrastructureCases = []ecosystemDomainExpansionTestCase{
	{
		name:    "defaults ecosystem includes basic infrastructure",
		allowed: []string{"defaults"},
		expected: []string{
			"crl3.digicert.com",
			"json-schema.org",
			"archive.ubuntu.com",
			"packagecloud.io",
			"packages.microsoft.com",
		},
		excluded: []string{
			"ghcr.io",
			"nuget.org",
			"github.com",
			"golang.org",
			"npmjs.org",
			"pypi.org",
		},
		context: "defaults",
	},
	{
		name:    "containers ecosystem includes container registries",
		allowed: []string{"containers"},
		expected: []string{
			"ghcr.io",
			"registry.hub.docker.com",
			"*.docker.io",
			"quay.io",
			"gcr.io",
		},
		context: "containers ecosystem",
	},
	{
		name:    "github ecosystem includes GitHub domains",
		allowed: []string{"github"},
		expected: []string{
			"*.githubusercontent.com",
			"raw.githubusercontent.com",
			"objects.githubusercontent.com",
			"patch-diff.githubusercontent.com",
			"lfs.github.com",
			"github.githubassets.com",
		},
		context: "github ecosystem",
	},
	{
		name:    "bazel ecosystem includes Bazel registry and download domains",
		allowed: []string{"bazel"},
		expected: []string{
			"releases.bazel.build",
			"mirror.bazel.build",
			"bcr.bazel.build",
		},
		context: "bazel ecosystem",
	},
}

var ecosystemDomainLanguageCases = []ecosystemDomainExpansionTestCase{
	{
		name:     "dotnet ecosystem includes .NET and NuGet domains",
		allowed:  []string{"dotnet"},
		expected: []string{"nuget.org", "dist.nuget.org", "api.nuget.org", "dotnet.microsoft.com", "dot.net"},
		context:  "dotnet ecosystem",
	},
	{
		name:     "python ecosystem includes Python package domains",
		allowed:  []string{"python"},
		expected: []string{"pypi.org", "pip.pypa.io", "*.pythonhosted.org", "files.pythonhosted.org", "anaconda.org"},
		context:  "python ecosystem",
	},
	{
		name:     "go ecosystem includes Go package domains",
		allowed:  []string{"go"},
		expected: []string{"go.dev", "golang.org", "proxy.golang.org", "sum.golang.org", "pkg.go.dev", "storage.googleapis.com"},
		context:  "go ecosystem",
	},
	{
		name:    "java ecosystem includes Java package and tooling domains",
		allowed: []string{"java"},
		expected: []string{
			"repo.maven.apache.org", "repo1.maven.org", "services.gradle.org", "plugins.gradle.org",
			"download.oracle.com", "dlcdn.apache.org", "archive.apache.org", "download.java.net",
			"api.foojay.io", "cdn.azul.com", "central.sonatype.com", "maven.google.com",
			"dl.google.com", "repo.gradle.org", "maven-central.storage-download.googleapis.com", "repository.apache.org",
		},
		context: "java ecosystem",
	},
	{
		name:    "node ecosystem includes Node.js package domains",
		allowed: []string{"node"},
		expected: []string{
			"npmjs.org", "registry.npmjs.com", "nodejs.org", "yarnpkg.com", "bun.sh", "deno.land",
			"jsr.io", "esm.sh", "googleapis.deno.dev", "googlechromelabs.github.io", "cdn.jsdelivr.net",
		},
		context: "node ecosystem",
	},
	{
		name:     "python-native ecosystem includes Rust FFI domains for native packages",
		allowed:  []string{"python-native"},
		expected: []string{"pypi.org", "pip.pypa.io", "crates.io", "index.crates.io", "static.crates.io"},
		context:  "python-native ecosystem",
	},
	{
		name:     "python ecosystem does not include crates domains",
		allowed:  []string{"python"},
		excluded: []string{"crates.io", "index.crates.io", "static.crates.io"},
		context:  "python ecosystem",
	},
	{
		name:     "julia ecosystem includes Julia package registry domains",
		allowed:  []string{"julia"},
		expected: []string{"pkg.julialang.org", "julialang.org"},
		context:  "julia ecosystem",
	},
	{
		name:     "lua ecosystem includes LuaRocks domains",
		allowed:  []string{"lua"},
		expected: []string{"luarocks.org", "www.luarocks.org"},
		context:  "lua ecosystem",
	},
	{
		name:    "latex ecosystem includes CTAN and TeX Live domains",
		allowed: []string{"latex"},
		expected: []string{
			"ctan.org", "mirror.ctan.org", "mirrors.ctan.org", "tug.org", "www.tug.org",
			"ftp.tug.org", "latex-project.org", "www.latex-project.org", "miktex.org", "packages.miktex.org",
		},
		context: "latex ecosystem",
	},
	{
		name:     "ocaml ecosystem includes opam domains",
		allowed:  []string{"ocaml"},
		expected: []string{"opam.ocaml.org", "ocaml.org", "erratique.ch"},
		context:  "ocaml ecosystem",
	},
	{
		name:     "r ecosystem includes CRAN domains",
		allowed:  []string{"r"},
		expected: []string{"cloud.r-project.org", "cran.r-project.org", "cran.rstudio.com"},
		context:  "r ecosystem",
	},
	{
		name:     "kotlin ecosystem includes JetBrains and Kotlin domains",
		allowed:  []string{"kotlin"},
		expected: []string{"download.jetbrains.com", "ge.jetbrains.com", "packages.jetbrains.team", "kotlin.bintray.com", "maven.pkg.jetbrains.space"},
		context:  "kotlin ecosystem",
	},
}

var ecosystemDomainCombinationCases = []ecosystemDomainExpansionTestCase{
	{
		name:    "multiple ecosystems can be combined",
		allowed: []string{"defaults", "dotnet", "python", "example.com"},
		expected: []string{
			"json-schema.org", "archive.ubuntu.com", "nuget.org", "dotnet.microsoft.com",
			"pypi.org", "*.pythonhosted.org", "example.com",
		},
		context: "combined ecosystems",
	},
	{
		name:          "unknown ecosystem identifier is treated as domain",
		allowed:       []string{"unknown-ecosystem", "example.com"},
		expected:      []string{"unknown-ecosystem", "example.com"},
		expectedCount: 2,
		context:       "literal domain",
	},
}

func TestEcosystemDomainExpansion(t *testing.T) {
	t.Run("infrastructure ecosystems", func(t *testing.T) {
		runEcosystemDomainExpansionCases(t, ecosystemDomainInfrastructureCases)
	})
	t.Run("language ecosystems", func(t *testing.T) {
		runEcosystemDomainExpansionCases(t, ecosystemDomainLanguageCases)
	})
	t.Run("combined ecosystems", func(t *testing.T) {
		runEcosystemDomainExpansionCases(t, ecosystemDomainCombinationCases)
	})
}

func runEcosystemDomainExpansionCases(t *testing.T, tests []ecosystemDomainExpansionTestCase) {
	t.Helper()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assertEcosystemDomains(t, tt)
		})
	}
}

func assertEcosystemDomains(t *testing.T, tt ecosystemDomainExpansionTestCase) {
	t.Helper()

	domains := GetAllowedDomains(&NetworkPermissions{Allowed: tt.allowed})
	if tt.expectedCount > 0 && len(domains) != tt.expectedCount {
		t.Fatalf("Expected %d domains, got %d: %v", tt.expectedCount, len(domains), domains)
	}

	for _, expectedDomain := range tt.expected {
		if !slices.Contains(domains, expectedDomain) {
			t.Errorf("Expected domain '%s' to be included in %s, but it was not found", expectedDomain, tt.context)
		}
	}

	for _, excludedDomain := range tt.excluded {
		if slices.Contains(domains, excludedDomain) {
			t.Errorf("Expected domain '%s' to NOT be included in %s, but it was found", excludedDomain, tt.context)
		}
	}
}

func TestAllEcosystemDomainFunctions(t *testing.T) {
	// Test that all ecosystem categories return non-empty slices
	ecosystemCategories := []string{
		"defaults", "containers", "bazel", "dotnet", "dart", "github", "go",
		"terraform", "haskell", "java", "julia", "kotlin", "latex", "linux-distros", "lua", "node",
		"ocaml", "perl", "php", "playwright", "python", "python-native", "r", "ruby", "rust", "swift",
	}

	for _, category := range ecosystemCategories {
		t.Run("getEcosystemDomains_"+category, func(t *testing.T) {
			domains := getEcosystemDomains(category)
			if len(domains) == 0 {
				t.Errorf("getEcosystemDomains(%q) returned empty slice, expected at least one domain", category)
			}

			// Check that all domains are non-empty strings
			for i, domain := range domains {
				if domain == "" {
					t.Errorf("getEcosystemDomains(%q) returned empty domain at index %d", category, i)
				}
			}
		})
	}
}

func TestEcosystemDomainsUniqueness(t *testing.T) {
	// Test that each ecosystem category returns unique domains (no duplicates)
	ecosystemCategories := []string{
		"defaults", "containers", "bazel", "dotnet", "dart", "github", "go",
		"terraform", "haskell", "java", "julia", "kotlin", "latex", "linux-distros", "lua", "node",
		"ocaml", "perl", "php", "playwright", "python", "python-native", "r", "ruby", "rust", "swift",
	}

	for _, category := range ecosystemCategories {
		t.Run("getEcosystemDomains_"+category+"_uniqueness", func(t *testing.T) {
			domains := getEcosystemDomains(category)
			seen := make(map[string]bool)

			for _, domain := range domains {
				if seen[domain] {
					t.Errorf("getEcosystemDomains(%q) returned duplicate domain: %s", category, domain)
				}
				seen[domain] = true
			}
		})
	}
}
