package workflow

import "github.com/github/gh-aw/pkg/constants"

// awfSupportsExcludeEnv returns true when the effective AWF version supports --exclude-env
// (introduced in AWF v0.25.3).
func awfSupportsExcludeEnv(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFExcludeEnvMinVersion)
}

// awfVersionAtLeast returns true when the effective AWF version is at or above minVersion.
//
// If firewallConfig has no version set, DefaultFirewallVersion is used. "latest" always
// returns true. Non-semver strings (e.g. branch names) return false (conservative).
func awfVersionAtLeast(firewallConfig *FirewallConfig, minVersion constants.Version) bool {
	var versionStr string
	if firewallConfig != nil && firewallConfig.Version != "" {
		versionStr = firewallConfig.Version
	}
	return versionAtLeast(versionStr, string(constants.DefaultFirewallVersion), string(minVersion))
}

// awfSupportsCliProxy returns true when the effective AWF version supports --difc-proxy-host
// and --difc-proxy-ca-cert (introduced in AWF v0.26.0).
func awfSupportsCliProxy(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFCliProxyMinVersion)
}

// awfSupportsAllowHostPorts returns true when the effective AWF version supports
// --allow-host-ports.
func awfSupportsAllowHostPorts(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFAllowHostPortsMinVersion)
}

// awfSupportsDockerHostPathPrefix returns true when the effective AWF version supports
// --docker-host-path-prefix.
func awfSupportsDockerHostPathPrefix(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFDockerHostPathPrefixMinVersion)
}

// awfSupportsTokenSteering returns true when the effective AWF version supports
// apiProxy.enableTokenSteering.
func awfSupportsTokenSteering(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFTokenSteeringMinVersion)
}

// awfSupportsChrootConfig returns true when the effective AWF version supports
// chroot.binariesSourcePath and chroot.identity.* in the config file (AWF v0.27.1+).
func awfSupportsChrootConfig(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFChrootConfigMinVersion)
}

// awfSupportsContainerRuntime returns true when the effective AWF version supports the
// containerRuntime field in the container config (gh-aw-firewall#6093).
// The field must not be emitted for older versions that do not recognise it.
func awfSupportsContainerRuntime(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFContainerRuntimeMinVersion)
}

// awfSupportsLegacySecurity returns true when the effective AWF version supports the
// --legacy-security flag (v0.27.32+). Older versions default to legacy mode and do not
// recognize this flag.
func awfSupportsLegacySecurity(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFLegacySecurityMinVersion)
}

// awfSupportsAPIProxyProviders returns true when the effective AWF version supports
// apiProxy.providers in awf-config.json.
func awfSupportsAPIProxyProviders(firewallConfig *FirewallConfig) bool {
	return awfVersionAtLeast(firewallConfig, constants.AWFAPIProxyProvidersMinVersion)
}
