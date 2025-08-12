## New Issue Link Creator

When suggesting that a user open a new GitHub issue, provide them with a clickable link that pre-fills the issue title and body. This makes it easier for users to create properly formatted issues.

### Format
Generate GitHub new issue URLs using this format:
```
https://github.com/${{ github.repository }}/issues/new?title=ENCODED_TITLE&body=ENCODED_BODY
```

### URL Encoding
- **Title**: URL-encode the suggested issue title
- **Body**: URL-encode the suggested issue body content
- Use proper URL encoding (spaces become `%20`, etc.)

### Example Usage
When recommending a user open an issue, format it like this:
```markdown
[📝 Open new issue: "Brief descriptive title"](https://github.com/${{ github.repository }}/issues/new?title=Brief%20descriptive%20title&body=Please%20describe%20the%20issue%20here...)
```

### When to Use
- When triaging reveals a legitimate bug that needs separate tracking
- When suggesting feature requests based on user feedback  
- When recommending documentation improvements
- When identifying reproducible issues that need developer attention

### Best Practices
- Keep titles concise but descriptive
- Include relevant context in the body
- Add appropriate labels or mention relevant areas of the codebase in the body
- Ensure the pre-filled content provides enough information for developers to understand and act on the issue