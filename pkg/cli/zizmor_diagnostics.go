package cli

import (
	"errors"
	"strconv"
	"strings"

	"github.com/github/gh-aw/pkg/scanfindings"
)

type zizmorRouteSegment struct {
	Key   *string `json:"Key"`
	Index *int    `json:"Index"`
}

func (location zizmorLocation) file() string {
	if path := location.Symbolic.Key.Local.GivenPath; path != "" {
		return path
	}
	return location.Symbolic.Key.Local.VerbatimPath
}

func (location zizmorLocation) scope() (string, error) {
	if location.Symbolic.Route == nil {
		return "", nil
	}
	if location.Symbolic.Route.Route == nil {
		return "", errors.New("zizmor returned an incomplete diagnostic scope")
	}
	var path strings.Builder
	path.WriteString("$")
	for _, segment := range location.Symbolic.Route.Route {
		switch {
		case segment.Key != nil && segment.Index == nil:
			path.WriteString(strings.TrimPrefix(scannerChildPath("$", *segment.Key), "$"))
		case segment.Index != nil && segment.Key == nil && *segment.Index >= 0:
			path.WriteString("[")
			path.WriteString(strconv.Itoa(*segment.Index))
			path.WriteString("]")
		default:
			return "", errors.New("zizmor returned an invalid diagnostic scope")
		}
	}
	return path.String(), nil
}

func validateZizmorDiagnostics(findings []zizmorFinding) error {
	for _, finding := range findings {
		if finding.Ident == "" || len(finding.Locations) == 0 {
			return errors.New("zizmor returned a finding without a rule or usable locations")
		}
		for index := range finding.Locations {
			location := &finding.Locations[index]
			point := location.Concrete.Location.StartPoint
			if location.file() == "" || point.Row < 0 || point.Column < 0 {
				return errors.New("zizmor returned a finding with an invalid file location")
			}
			path, err := location.scope()
			if err != nil {
				return err
			}
			location.path = path
		}
	}
	return nil
}

func zizmorDisplayLocation(file string, locations []zizmorLocation) (zizmorLocation, bool) {
	var selected zizmorLocation
	found := false
	for _, location := range locations {
		if location.file() == file && (!found || location.Symbolic.Kind == "Primary") {
			selected = location
			found = true
		}
	}
	return selected, found
}

func zizmorDiagnosticsToShared(file string, findings []zizmorFinding, lines []string) []scanfindings.Finding {
	var diagnostics []scanfindings.Finding
	for _, finding := range findings {
		selected, found := zizmorDisplayLocation(file, finding.Locations)
		if !found {
			continue
		}
		display := finding
		display.Locations = []zizmorLocation{selected}
		for _, shared := range zizmorFindingsToShared(file, []zizmorFinding{display}, lines) {
			for _, location := range finding.Locations {
				shared.Locations = append(shared.Locations, scanfindings.SourceLocation{
					File: location.file(), Path: location.path, Role: location.Symbolic.Kind,
					Line: location.Concrete.Location.StartPoint.Row + 1, Column: location.Concrete.Location.StartPoint.Column + 1,
				})
			}
			diagnostics = append(diagnostics, shared)
		}
	}
	return diagnostics
}
