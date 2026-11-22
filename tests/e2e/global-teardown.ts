import { resetTestUsers } from "./helpers";

/** Removes the E2E users and everything they created (cascade). */
export default async function globalTeardown(): Promise<void> {
  resetTestUsers();
}
