#!/usr/bin/env node
// @ts-check

/**
 * Unit tests for the AW wizard shared JSON data model validation
 * (docs/src/lib/wizard/validation.ts).
 *
 * These tests exercise `validateModel` directly so malformed wizard data
 * model JSON is caught automatically instead of only failing at Astro
 * build/import time.
 *
 * Run with: node docs/src/lib/wizard/model.test.js
 */

import { validateModel } from './validation.ts';
import validModel from '../../data/wizard-data-model.json' with { type: 'json' };

let passed = 0;
let failed = 0;

function assertThrows(fn, label) {
	try {
		fn();
		console.error(`  ✗ ${label}`);
		console.error('    expected: function to throw');
		console.error('    actual:   no error was thrown');
		failed++;
	} catch (error) {
		console.log(`  ✓ ${label} (${/** @type {Error} */ (error).message})`);
		passed++;
	}
}

function assertDoesNotThrow(fn, label) {
	try {
		fn();
		console.log(`  ✓ ${label}`);
		passed++;
	} catch (error) {
		console.error(`  ✗ ${label}`);
		console.error(`    expected: function not to throw`);
		console.error(`    actual:   threw ${/** @type {Error} */ (error).message}`);
		failed++;
	}
}

function clone(data) {
	return JSON.parse(JSON.stringify(data));
}

// -------------------------------------------------------------------
// Test: the real, checked-in wizard data model is valid
// -------------------------------------------------------------------
console.log('\nreal wizard-data-model.json:');
{
	assertDoesNotThrow(() => validateModel(clone(validModel)), 'checked-in wizard-data-model.json passes validation');
}

// -------------------------------------------------------------------
// Test: non-object input is rejected
// -------------------------------------------------------------------
console.log('\nnon-object input:');
{
	assertThrows(() => validateModel(null), 'null is rejected');
	assertThrows(() => validateModel('not an object'), 'string is rejected');
	assertThrows(() => validateModel([]), 'array is rejected');
}

// -------------------------------------------------------------------
// Test: missing/invalid version field
// -------------------------------------------------------------------
console.log('\nversion field:');
{
	const missingVersion = clone(validModel);
	delete missingVersion.version;
	assertThrows(() => validateModel(missingVersion), 'missing version is rejected');

	const nonSemverVersion = clone(validModel);
	nonSemverVersion.version = '1.0';
	assertThrows(() => validateModel(nonSemverVersion), 'non-semver version ("1.0") is rejected');
}

// -------------------------------------------------------------------
// Test: empty top-level arrays are rejected
// -------------------------------------------------------------------
console.log('\nempty top-level arrays:');
{
	const emptyGoals = clone(validModel);
	emptyGoals.goalCategories = [];
	assertThrows(() => validateModel(emptyGoals), 'empty goalCategories is rejected');

	const emptyTriggers = clone(validModel);
	emptyTriggers.triggerOptions = [];
	assertThrows(() => validateModel(emptyTriggers), 'empty triggerOptions is rejected');

	const emptyDestinations = clone(validModel);
	emptyDestinations.destinationOptions = [];
	assertThrows(() => validateModel(emptyDestinations), 'empty destinationOptions is rejected');
}

// -------------------------------------------------------------------
// Test: goal category missing required fields
// -------------------------------------------------------------------
console.log('\ngoal category shape:');
{
	const missingTriggerIds = clone(validModel);
	delete missingTriggerIds.goalCategories[0].triggerOptionIds;
	assertThrows(() => validateModel(missingTriggerIds), 'goal category missing triggerOptionIds is rejected');

	const missingDestinationIds = clone(validModel);
	delete missingDestinationIds.goalCategories[0].destinationOptionIds;
	assertThrows(() => validateModel(missingDestinationIds), 'goal category missing destinationOptionIds is rejected');

	const missingGoalText = clone(validModel);
	delete missingGoalText.goalCategories[0].text;
	assertThrows(() => validateModel(missingGoalText), 'goal category missing text.label is rejected');
}

// -------------------------------------------------------------------
// Test: trigger/destination option missing required fields
// -------------------------------------------------------------------
console.log('\ntrigger/destination option shape:');
{
	const missingTriggerType = clone(validModel);
	delete missingTriggerType.triggerOptions[0].type;
	assertThrows(() => validateModel(missingTriggerType), 'trigger option missing type is rejected');

	const missingDestinationSafeOutputType = clone(validModel);
	delete missingDestinationSafeOutputType.destinationOptions[0].safeOutputType;
	assertThrows(
		() => validateModel(missingDestinationSafeOutputType),
		'destination option missing safeOutputType is rejected',
	);
}

// -------------------------------------------------------------------
// Summary
// -------------------------------------------------------------------
console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
