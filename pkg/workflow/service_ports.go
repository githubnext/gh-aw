// This file provides helper functions for extracting service port mappings from
// GitHub Actions services: configuration and generating ${{ job.services.<id>.ports['<port>'] }}
// expressions for AWF's --allow-host-service-ports flag.
//
// When a workflow uses GitHub Actions services: with port mappings (e.g., PostgreSQL, Redis),
// the compiled workflow runs the agent inside AWF's isolated network. The agent cannot reach
// service containers without explicit --allow-host-service-ports configuration. This file
// automatically detects service ports and generates the necessary expressions.

package workflow

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	goyaml "gopkg.in/yaml.v3"
)

var servicePortsLog = logger.New("workflow:service_ports")

// maxPortRangeExpansion is the maximum number of ports to expand from a range specification.
// This prevents accidentally generating thousands of expressions from a large port range.
const maxPortRangeExpansion = 32

// ExtractServicePortExpressions parses the services: YAML string from WorkflowData.Services
// and returns a comma-separated string of ${{ job.services.<id>.ports['<port>'] }} expressions
// for all TCP port mappings found.
//
// The returned string is suitable for passing as --allow-host-service-ports to AWF.
// Returns empty string if no services or no port mappings are found.
//
// Parameters:
//   - servicesYAML: Raw YAML string from WorkflowData.Services (includes "services:" wrapper)
//
// Returns:
//   - expressions: Comma-separated ${{ }} expressions for all service ports
//   - warnings: Any warnings generated during parsing (e.g., UDP ports, services without ports)
func ExtractServicePortExpressions(servicesYAML string) (string, []string) {
	if servicesYAML == "" {
		return "", nil
	}

	servicePortsLog.Print("Extracting service port expressions from services YAML")

	// Parse the services YAML
	var wrapper map[string]any
	if err := goyaml.Unmarshal([]byte(servicesYAML), &wrapper); err != nil {
		servicePortsLog.Printf("Failed to parse services YAML: %v", err)
		return "", nil
	}

	services, ok := wrapper["services"].(map[string]any)
	if !ok {
		servicePortsLog.Print("No services map found in YAML")
		return "", nil
	}

	var expressions []string
	var warnings []string

	// Sort service IDs for deterministic output
	serviceIDs := make([]string, 0, len(services))
	for id := range services {
		serviceIDs = append(serviceIDs, id)
	}
	sort.Strings(serviceIDs)

	for _, serviceID := range serviceIDs {
		serviceConfig := services[serviceID]
		svcMap, ok := serviceConfig.(map[string]any)
		if !ok {
			servicePortsLog.Printf("Service %s is not a map, skipping", serviceID)
			continue
		}

		ports, hasPorts := svcMap["ports"]
		if !hasPorts {
			warnings = append(warnings, fmt.Sprintf("service %q has no ports mapping; it will not be reachable from the AWF sandbox", serviceID))
			servicePortsLog.Printf("Service %s has no ports, skipping", serviceID)
			continue
		}

		portsList, ok := ports.([]any)
		if !ok {
			servicePortsLog.Printf("Service %s ports is not a list, skipping", serviceID)
			continue
		}

		for _, portSpec := range portsList {
			containerPorts, portWarnings := parsePortSpec(portSpec)
			for _, w := range portWarnings {
				warnings = append(warnings, fmt.Sprintf("service %q: %s", serviceID, w))
			}
			for _, cp := range containerPorts {
				expr := fmt.Sprintf("${{ job.services.%s.ports['%d'] }}", serviceID, cp)
				expressions = append(expressions, expr)
			}
		}
	}

	if len(expressions) == 0 {
		servicePortsLog.Print("No service port expressions generated")
		return "", warnings
	}

	result := strings.Join(expressions, ",")
	servicePortsLog.Printf("Generated %d service port expressions", len(expressions))
	return result, warnings
}

// parsePortSpec parses a single port specification and returns the container port(s).
// Supports formats:
//   - "5432:5432" (host:container)
//   - "5432" (container only, dynamic host port)
//   - "49152:5432" (remapped host port)
//   - "5432/tcp" (explicit TCP)
//   - "5432/udp" (skipped with warning)
//   - "6000-6010:6000-6010" (range)
//   - 5432 (integer)
//
// Returns container port numbers and any warnings.
func parsePortSpec(spec any) ([]int, []string) {
	var portStr string
	switch v := spec.(type) {
	case int:
		return []int{v}, nil
	case float64:
		// YAML may parse unquoted numbers as float64
		return []int{int(v)}, nil
	case string:
		portStr = v
	default:
		return nil, []string{fmt.Sprintf("unsupported port spec type %T: %v", spec, spec)}
	}

	portStr = strings.TrimSpace(portStr)
	if portStr == "" {
		return nil, nil
	}

	// Check for protocol suffix
	protocol := "tcp"
	if idx := strings.LastIndex(portStr, "/"); idx != -1 {
		protocol = strings.ToLower(portStr[idx+1:])
		portStr = portStr[:idx]
	}

	if protocol == "udp" {
		return nil, []string{fmt.Sprintf("UDP port %q skipped; AWF only supports TCP", portStr)}
	}

	// Split host:container
	var containerPart string
	if idx := strings.Index(portStr, ":"); idx != -1 {
		containerPart = portStr[idx+1:]
	} else {
		containerPart = portStr
	}

	// Check for port range (e.g., "6000-6010")
	if idx := strings.Index(containerPart, "-"); idx != -1 {
		startStr := containerPart[:idx]
		endStr := containerPart[idx+1:]

		start, err1 := strconv.Atoi(startStr)
		end, err2 := strconv.Atoi(endStr)
		if err1 != nil || err2 != nil {
			return nil, []string{fmt.Sprintf("invalid port range %q", containerPart)}
		}

		if end < start {
			return nil, []string{fmt.Sprintf("invalid port range %q: end < start", containerPart)}
		}

		count := end - start + 1
		if count > maxPortRangeExpansion {
			return nil, []string{fmt.Sprintf("port range %q expands to %d ports, exceeding cap of %d", containerPart, count, maxPortRangeExpansion)}
		}

		ports := make([]int, 0, count)
		for p := start; p <= end; p++ {
			ports = append(ports, p)
		}
		return ports, nil
	}

	// Single port
	port, err := strconv.Atoi(containerPart)
	if err != nil {
		return nil, []string{fmt.Sprintf("invalid port number %q", containerPart)}
	}

	return []int{port}, nil
}
