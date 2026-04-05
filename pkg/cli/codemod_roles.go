package cli

import (
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var rolesCodemodLog = logger.New("cli:codemod_roles")

// getRolesToOnRolesCodemod creates a codemod for moving top-level 'roles' to 'on.roles'
func getRolesToOnRolesCodemod() Codemod {
	return newMoveTopLevelKeyToOnBlockCodemod(moveToOnBlockConfig{
		ID:           "roles-to-on-roles",
		Name:         "Move roles to on.roles",
		Description:  "Moves the top-level 'roles' field to 'on.roles' as per the new frontmatter structure",
		IntroducedIn: "0.10.0",
		FieldKey:     "roles",
		IsInlineSingle: func(v string) bool {
			return v == "all" || strings.HasPrefix(v, "[")
		},
		Log: rolesCodemodLog,
	})
}
