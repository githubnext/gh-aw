// @ts-check
/**
 * Example JavaScript handler for safe-inputs
 * Greets a person by name
 */
async function execute(inputs) {
  const { name } = inputs || {};
  
  if (!name) {
    throw new Error("Name is required");
  }
  
  return {
    greeting: `Hello, ${name}! Welcome to MCP Gateway with safe-inputs.`,
    timestamp: new Date().toISOString()
  };
}

module.exports = { execute };
