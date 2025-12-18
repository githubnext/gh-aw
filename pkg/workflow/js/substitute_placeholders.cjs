const fs = require("fs"),
  substitutePlaceholders = async ({ file, substitutions }) => {
    if (!file) throw new Error("file parameter is required");
    if (!substitutions || "object" != typeof substitutions) throw new Error("substitutions parameter must be an object");
    let content;
    try {
      content = fs.readFileSync(file, "utf8");
    } catch (error) {
      throw new Error(`Failed to read file ${file}: ${error.message}`);
    }
    for (const [key, value] of Object.entries(substitutions)) {
      const placeholder = `__${key}__`;
      content = content.split(placeholder).join(value);
    }
    try {
      fs.writeFileSync(file, content, "utf8");
    } catch (error) {
      throw new Error(`Failed to write file ${file}: ${error.message}`);
    }
    return `Successfully substituted ${Object.keys(substitutions).length} placeholder(s) in ${file}`;
  };
module.exports = substitutePlaceholders;
