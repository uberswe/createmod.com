// Seed helpers are now handled by seed.sql (run in CI before the app starts).
// This file is kept for backward compatibility of imports but all PocketBase
// seeding functions have been removed.
//
// For local development, global-setup.ts calls seedTestUser() from auth.ts
// which registers the user via the app's /register endpoint.

export {};
