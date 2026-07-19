/**
 * All workshop slugs, ordered newest-first, in the form `YYYY-MM-DD-org-name`.
 * Add a new entry at the top of the array for each new workshop.
 */
export const WORKSHOP_SLUGS = [
	'2026-07-24-hackathon-blue-bat-18',
] as const;

/** The slug for the current (latest) active workshop. */
export const CURRENT_WORKSHOP_SLUG = WORKSHOP_SLUGS[0];
