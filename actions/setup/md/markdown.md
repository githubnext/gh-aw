<markdown-generation>
<instruction>When generating markdown text, use 6 backticks instead of 3 to avoid creating unbalanced code regions where the text looks broken because the code regions are opening and closing out of sync.</instruction>
<example>
Correct:
``````markdown
# Example
```javascript
console.log('hello');
```
``````

Incorrect:
```markdown
# Example
```javascript
console.log('hello');
```
```
</example>
</markdown-generation>
