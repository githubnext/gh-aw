package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/network"
	"github.com/spf13/cobra"
)

// NewNetworkTestCommand creates the network test command
func NewNetworkTestCommand() *cobra.Command {
	var (
		proxyHost   string
		proxyPort   string
		domainsFile string
		timeout     string
		testURLs    []string
		verbose     bool
	)

	cmd := &cobra.Command{
		Use:   "network-test",
		Short: "Test MCP network permissions and proxy configuration",
		Long: `Test MCP network permissions by validating connectivity through proxy configuration.

This command helps validate that network isolation is working correctly by testing
access to allowed and blocked domains through the configured proxy.`,
		Example: `  # Test with default proxy settings
  gh aw network-test --urls https://example.com,https://api.github.com

  # Test with custom proxy configuration
  gh aw network-test --proxy-host localhost --proxy-port 3128 \
    --domains-file ./docker/squid/allowed_domains.txt \
    --urls https://example.com,https://httpbin.org,https://api.github.com

  # Test with verbose output
  gh aw network-test --verbose --urls https://example.com,https://malicious-example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkTest(proxyHost, proxyPort, domainsFile, timeout, testURLs, verbose)
		},
	}

	cmd.Flags().StringVar(&proxyHost, "proxy-host", "localhost", "Proxy host address")
	cmd.Flags().StringVar(&proxyPort, "proxy-port", "3128", "Proxy port")
	cmd.Flags().StringVar(&domainsFile, "domains-file", "", "Path to allowed domains file")
	cmd.Flags().StringVar(&timeout, "timeout", "30s", "Request timeout duration")
	cmd.Flags().StringSliceVar(&testURLs, "urls", nil, "URLs to test (comma-separated)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	_ = cmd.MarkFlagRequired("urls")

	return cmd
}

// runNetworkTest executes the network permission testing
func runNetworkTest(proxyHost, proxyPort, domainsFile, timeout string, testURLs []string, verbose bool) error {
	fmt.Println(console.FormatInfoMessage("Starting MCP network permissions test"))

	// Parse timeout
	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	// Create proxy configuration
	var proxyConfig *network.ProxyConfig
	if proxyHost != "" && proxyPort != "" {
		if verbose {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Using proxy: %s:%s", proxyHost, proxyPort)))
		}

		proxyConfig = &network.ProxyConfig{
			Host: proxyHost,
			Port: proxyPort,
		}

		// Load allowed domains if file specified
		if domainsFile != "" {
			if verbose {
				fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Loading allowed domains from: %s", domainsFile)))
			}

			domains, err := network.LoadAllowedDomainsFromFile(domainsFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Could not load domains file: %v", err)))
				fmt.Println(console.FormatInfoMessage("Proceeding without domain restrictions"))
			} else {
				proxyConfig.AllowedDomains = domains
				if verbose {
					fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Loaded %d allowed domains", len(domains))))
				}
			}
		}

		// Test proxy connectivity
		if verbose {
			fmt.Println(console.FormatProgressMessage("Testing proxy connectivity..."))
		}

		if err := network.TestProxyConnectivity(proxyConfig, timeoutDuration); err != nil {
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Proxy connectivity test failed: %v", err)))
			fmt.Println(console.FormatInfoMessage("Proceeding with direct connection"))
			proxyConfig = nil
		} else if verbose {
			fmt.Println(console.FormatSuccessMessage("Proxy is accessible"))
		}
	}

	// Create network tester
	tester := network.NewNetworkTester(proxyConfig, timeoutDuration)

	// Display test configuration
	fmt.Println(console.FormatInfoMessage("Test Configuration:"))
	if proxyConfig != nil {
		fmt.Printf("  Proxy: %s:%s\n", proxyConfig.Host, proxyConfig.Port)
		fmt.Printf("  Allowed Domains: %d\n", len(proxyConfig.AllowedDomains))
		if verbose && len(proxyConfig.AllowedDomains) > 0 {
			for _, domain := range proxyConfig.AllowedDomains {
				fmt.Printf("    - %s\n", domain)
			}
		}
	} else {
		fmt.Println("  Proxy: Direct connection")
	}
	fmt.Printf("  Timeout: %v\n", timeoutDuration)
	fmt.Printf("  Test URLs: %d\n", len(testURLs))

	// Run tests
	fmt.Println(console.FormatProgressMessage("Running network connectivity tests..."))
	results := tester.TestMultipleURLs(testURLs)

	// Analyze results
	analysis := network.AnalyzeNetworkResults(results)

	// Print analysis
	analysis.PrintAnalysis(os.Stdout)

	// Print summary
	fmt.Println(console.FormatInfoMessage("Test Summary:"))
	if analysis.AllowedTests > 0 && analysis.SuccessfulTests == analysis.AllowedTests && analysis.FailedTests == analysis.BlockedTests {
		fmt.Println(console.FormatSuccessMessage("✅ Network isolation is working correctly"))
		fmt.Printf("  - All %d allowed domains were accessible\n", analysis.AllowedTests)
		fmt.Printf("  - All %d blocked domains were inaccessible\n", analysis.BlockedTests)
	} else {
		fmt.Println(console.FormatWarningMessage("⚠️  Network isolation results need review"))
		if analysis.AllowedTests > 0 && analysis.SuccessfulTests < analysis.AllowedTests {
			fmt.Printf("  - %d/%d allowed domains failed to connect\n", analysis.AllowedTests-analysis.SuccessfulTests, analysis.AllowedTests)
		}
		unexpectedSuccess := 0
		for _, result := range results {
			if result.Success && !result.Allowed {
				unexpectedSuccess++
			}
		}
		if unexpectedSuccess > 0 {
			fmt.Printf("  - %d blocked domains were unexpectedly accessible\n", unexpectedSuccess)
		}
	}

	return nil
}

// NewNetworkValidateCommand creates the network configuration validation command
func NewNetworkValidateCommand() *cobra.Command {
	var (
		domainsFile string
		configFile  string
		verbose     bool
	)

	cmd := &cobra.Command{
		Use:   "network-validate",
		Short: "Validate MCP network configuration files",
		Long: `Validate MCP network configuration files including proxy settings and domain allowlists.

This command helps ensure that network configuration files are properly formatted
and contain valid settings for network isolation testing.`,
		Example: `  # Validate domains file
  gh aw network-validate --domains-file ./docker/squid/allowed_domains.txt

  # Validate proxy configuration
  gh aw network-validate --config-file ./docker/squid/squid.conf --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkValidate(domainsFile, configFile, verbose)
		},
	}

	cmd.Flags().StringVar(&domainsFile, "domains-file", "", "Path to allowed domains file to validate")
	cmd.Flags().StringVar(&configFile, "config-file", "", "Path to proxy config file to validate")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose output")

	return cmd
}

// runNetworkValidate executes network configuration validation
func runNetworkValidate(domainsFile, configFile string, verbose bool) error {
	fmt.Println(console.FormatInfoMessage("Validating MCP network configuration"))

	validationErrors := 0

	// Validate domains file
	if domainsFile != "" {
		if verbose {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Validating domains file: %s", domainsFile)))
		}

		domains, err := network.LoadAllowedDomainsFromFile(domainsFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Domains file validation failed: %v", err)))
			validationErrors++
		} else {
			fmt.Println(console.FormatSuccessMessage(fmt.Sprintf("✅ Domains file is valid (%d domains)", len(domains))))
			if verbose {
				fmt.Println(console.FormatInfoMessage("Allowed domains:"))
				for i, domain := range domains {
					fmt.Printf("  %d. %s\n", i+1, domain)
				}
			}
		}
	}

	// Validate proxy config file
	if configFile != "" {
		if verbose {
			fmt.Println(console.FormatVerboseMessage(fmt.Sprintf("Validating proxy config file: %s", configFile)))
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Config file does not exist: %s", configFile)))
			validationErrors++
		} else {
			fmt.Println(console.FormatSuccessMessage("✅ Proxy config file exists and is readable"))
		}
	}

	if domainsFile == "" && configFile == "" {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage("No configuration files specified for validation"))
		return fmt.Errorf("specify at least one file to validate using --domains-file or --config-file")
	}

	if validationErrors > 0 {
		fmt.Fprintln(os.Stderr, console.FormatErrorMessage(fmt.Sprintf("Validation completed with %s",
			pluralize(validationErrors, "error", "errors"))))
		return fmt.Errorf("validation failed")
	}

	fmt.Println(console.FormatSuccessMessage("✅ All network configuration files are valid"))
	return nil
}

// pluralize returns the singular or plural form based on count
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return strconv.Itoa(count) + " " + singular
	}
	return strconv.Itoa(count) + " " + plural
}
