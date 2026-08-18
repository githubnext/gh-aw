export interface WizardText {
  label: string;
  help?: string;
}

export interface WizardGoalCategory {
  id: string;
  text: WizardText;
  triggerOptionIds: string[];
  destinationOptionIds: string[];
  defaultTriggerOptionId?: string;
  defaultDestinationOptionId?: string;
}

export interface WizardTriggerOption {
  id: string;
  type: string;
  text: WizardText;
  frontmatter?: Record<string, unknown>;
}

export interface WizardDestinationOption {
  id: string;
  safeOutputType: string;
  text: WizardText;
  inferFromTriggerTypes?: string[];
  frontmatter?: Record<string, unknown>;
}

export interface WizardPromptSection {
  id: string;
  heading: string;
}

export interface WizardDataModel {
  version: string;
  goalCategories: WizardGoalCategory[];
  triggerOptions: WizardTriggerOption[];
  destinationOptions: WizardDestinationOption[];
  promptTemplate?: {
    introText?: string;
    sections?: WizardPromptSection[];
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function hasText(value: unknown): value is WizardText {
  return isRecord(value) && typeof value.label === 'string' && (value.help === undefined || typeof value.help === 'string');
}

/**
 * Validates that `data` conforms to the shared wizard JSON data model shape,
 * throwing a descriptive Error on the first violation found. Used both at
 * module load time (via model.ts) and by automated tests to catch malformed
 * model data before it reaches the docs build.
 */
export function validateModel(data: unknown): asserts data is WizardDataModel {
  if (!isRecord(data)) throw new Error('Wizard data model must be an object.');
  if (typeof data.version !== 'string') throw new Error('Wizard data model version must be a string.');
  if (!/^\d+\.\d+\.\d+$/.test(data.version)) throw new Error('Wizard data model version must use semver.');
  if (!Array.isArray(data.goalCategories) || data.goalCategories.length === 0) {
    throw new Error('Wizard data model requires at least one goal category.');
  }
  if (!Array.isArray(data.triggerOptions) || data.triggerOptions.length === 0) {
    throw new Error('Wizard data model requires at least one trigger option.');
  }
  if (!Array.isArray(data.destinationOptions) || data.destinationOptions.length === 0) {
    throw new Error('Wizard data model requires at least one destination option.');
  }

  for (const goal of data.goalCategories) {
    if (!isRecord(goal) || typeof goal.id !== 'string' || !hasText(goal.text)) {
      throw new Error('Each wizard goal category must include id and text.label.');
    }
    if (!Array.isArray(goal.triggerOptionIds) || goal.triggerOptionIds.length === 0) {
      throw new Error(`Goal category "${goal.id}" must define triggerOptionIds.`);
    }
    if (!Array.isArray(goal.destinationOptionIds) || goal.destinationOptionIds.length === 0) {
      throw new Error(`Goal category "${goal.id}" must define destinationOptionIds.`);
    }
  }

  for (const trigger of data.triggerOptions) {
    if (!isRecord(trigger) || typeof trigger.id !== 'string' || typeof trigger.type !== 'string' || !hasText(trigger.text)) {
      throw new Error('Each wizard trigger option must include id, type, and text.label.');
    }
  }

  for (const destination of data.destinationOptions) {
    if (!isRecord(destination) || typeof destination.id !== 'string' || typeof destination.safeOutputType !== 'string' || !hasText(destination.text)) {
      throw new Error('Each wizard destination option must include id, safeOutputType, and text.label.');
    }
  }
}
