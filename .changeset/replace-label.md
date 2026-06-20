"gh-aw": minor

Add replace-label safe-outputs type for atomic label state transitions

The new `replace-label` safe output type removes one label and adds another in a single atomic GraphQL operation. It is designed for label-based state machines where removing a label transitions an issue or PR to the next state (e.g. `in-progress` → `done`).

Key capabilities:
- `allowed-add`: optional list of label patterns permitted as the target label
- `allowed-remove`: optional list of label patterns permitted as the source label  
- `blocked`: label patterns rejected for both add and remove operations
- Inherits `target`, `target-repo`, `allowed-repos`, `required-labels`, `required-title-prefix`, `staged`, `max`, and `github-token` from the shared config infrastructure
- Uses a combined GraphQL mutation (`removeLabelsFromLabelable` + `addLabelsToLabelable`) for a single-request state transition
- Automatically creates the target label in the repository if it does not already exist
- Gracefully handles the case where the label to remove is not present on the item (still adds the new label)
