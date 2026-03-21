To allow these resources, lower `min-integrity` in your GitHub frontmatter:

```yaml
tools:
  github:
    min-integrity: approved  # merged | approved | unapproved | none
```

See [Integrity Filtering](https://github.github.com/gh-aw/reference/integrity/) for more information.
