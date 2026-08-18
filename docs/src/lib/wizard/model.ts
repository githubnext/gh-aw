import modelData from '../../data/wizard-data-model.json';
import { validateModel, type WizardDataModel } from './validation';

export type {
  WizardText,
  WizardGoalCategory,
  WizardTriggerOption,
  WizardDestinationOption,
  WizardPromptSection,
  WizardDataModel,
} from './validation';

export { validateModel } from './validation';

validateModel(modelData);

export const wizardDataModel: WizardDataModel = modelData;
