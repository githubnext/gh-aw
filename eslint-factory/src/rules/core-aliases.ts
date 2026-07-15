// Known `@actions/core` binding names used across lint rules.
// Only exact known aliases are matched — broad prefix matching (for example `/^core/i`)
// would silently flag unrelated objects that happen to start with "core".
export const CORE_ALIASES = new Set(["core", "coreObj"]);
