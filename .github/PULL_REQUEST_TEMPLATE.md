## Description

<!-- Provide a brief description of the changes in this PR -->

## Related Issue(s)

<!-- Link to related issues using #issue-number -->

Fixes #

## Type of Change

<!-- Mark the relevant option with an 'x' -->

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test coverage improvement

## Testing

<!-- Describe the tests you ran and how to reproduce them -->

- [ ] I have tested these changes locally
- [ ] I have added/updated tests that prove my fix is effective or that my feature works
- [ ] All existing tests pass locally
- [ ] I have run `make agent-finish` and all checks pass

## Documentation

- [ ] I have updated relevant documentation (DEVGUIDE.md, SECURITY.md, README.md, etc.)
- [ ] I have added/updated code comments where necessary
- [ ] I have updated examples if applicable

## Security Checklist

<!-- For workflows that handle user-controlled data -->

- [ ] No `envsubst` usage on untrusted data (issue bodies, PR titles, comments, etc.)
- [ ] All user input treated as literal strings (passed through environment variables)
- [ ] Templates use placeholder tokens (e.g., `__VAR__`) if applicable
- [ ] Sed substitution includes proper escaping (e.g., `${VAR//|/\\|}`) if applicable
- [ ] No direct template expressions (`${{ }}`) in shell commands with untrusted data
- [ ] Security scans pass (`./gh-aw compile --zizmor --actionlint --poutine`)
- [ ] N/A - This PR does not involve workflow changes or user-controlled data

## Code Quality

- [ ] My code follows the style guidelines of this project
- [ ] I have performed a self-review of my own code
- [ ] I have run `make fmt` and `make lint` successfully
- [ ] My changes generate no new warnings or errors
- [ ] I have made minimal changes to accomplish the task

## Additional Notes

<!-- Add any additional context, screenshots, or information that reviewers should know -->

## Checklist

- [ ] I have read the [CONTRIBUTING](../CONTRIBUTING.md) guidelines
- [ ] I have reviewed the [DEVGUIDE](../DEVGUIDE.md) for best practices
- [ ] I understand the [SECURITY](../SECURITY.md) requirements
- [ ] This PR has a clear and descriptive title
- [ ] I have rebased my branch on the latest main (if needed)
