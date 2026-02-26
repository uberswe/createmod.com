import { pbURL } from './auth';
import * as fs from 'fs';
import * as path from 'path';

/**
 * Authenticate as PocketBase superuser.
 * Uses the default dev credentials from .env.example (CREATE_ADMIN=true).
 * Falls back to the PB_ADMIN_EMAIL / PB_ADMIN_PASSWORD env vars if set.
 */
export async function superuserToken(): Promise<string> {
  const pb = pbURL();
  const email = process.env.PB_ADMIN_EMAIL || 'local@createmod.com';
  const password = process.env.PB_ADMIN_PASSWORD || 'jfq.utb*jda2abg!WCR';

  const params = new URLSearchParams();
  params.set('identity', email);
  params.set('password', password);
  const resp = await fetch(`${pb}/api/collections/_superusers/auth-with-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: params.toString(),
  });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`superuser auth failed: ${resp.status} ${text}`);
  }

  const data = await resp.json();
  return data.token as string;
}

/**
 * Seed a test schematic via PocketBase admin API.
 *
 * Idempotent: if a schematic with name "e2e-test-schematic" already exists,
 * returns its id/name without creating a duplicate.
 *
 * The schematic is created as moderated + non-deleted so the download
 * interstitial and schematic page can find it.
 *
 * Returns { id, name } of the schematic record.
 */
export async function seedTestSchematic(
  authorId: string,
  token: string,
): Promise<{ id: string; name: string }> {
  const pb = pbURL();

  // Check if it already exists
  const listResp = await fetch(
    `${pb}/api/collections/schematics/records?filter=(name='e2e-test-schematic')`,
    { headers: { Authorization: token } },
  );

  if (listResp.ok) {
    const list = await listResp.json();
    if (list.items && list.items.length > 0) {
      return { id: list.items[0].id, name: list.items[0].name };
    }
  }

  // Build multipart form with files
  const form = new FormData();
  form.append('name', 'e2e-test-schematic');
  form.append('title', 'E2E Test Schematic');
  form.append('author', authorId);
  form.append('moderated', 'true');
  form.append('description', 'Schematic created by Playwright E2E global setup.');

  // Attach featured_image from testdata
  const imgPath = path.resolve(__dirname, '../../../testdata/image.png');
  if (fs.existsSync(imgPath)) {
    const imgBuf = fs.readFileSync(imgPath);
    form.append('featured_image', new Blob([imgBuf], { type: 'image/png' }), 'image.png');
  }

  // Attach a minimal .nbt file from fixtures
  const nbtPath = path.resolve(__dirname, '../../../tests/fixtures/sample.nbt');
  if (fs.existsSync(nbtPath)) {
    const nbtBuf = fs.readFileSync(nbtPath);
    form.append('schematic_file', new Blob([nbtBuf], { type: 'application/octet-stream' }), 'sample.nbt');
  }

  const createResp = await fetch(`${pb}/api/collections/schematics/records`, {
    method: 'POST',
    headers: { Authorization: token },
    body: form,
  });

  if (!createResp.ok) {
    const text = await createResp.text();
    throw new Error(`seedTestSchematic failed: ${createResp.status} ${text}`);
  }

  const rec = await createResp.json();
  return { id: rec.id, name: rec.name };
}

/**
 * Seed a test collection via PocketBase admin API.
 *
 * Idempotent: if a collection with slug "e2e-test-collection" already exists,
 * returns its id/slug without creating a duplicate.
 *
 * Returns { id, slug } of the collection record.
 */
export async function seedTestCollection(
  authorId: string,
  schematicIds: string[],
  token: string,
): Promise<{ id: string; slug: string }> {
  const pb = pbURL();

  // Check if it already exists
  const listResp = await fetch(
    `${pb}/api/collections/collections/records?filter=(slug='e2e-test-collection')`,
    { headers: { Authorization: token } },
  );

  if (listResp.ok) {
    const list = await listResp.json();
    if (list.items && list.items.length > 0) {
      return { id: list.items[0].id, slug: list.items[0].slug };
    }
  }

  const createResp = await fetch(`${pb}/api/collections/collections/records`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: token,
    },
    body: JSON.stringify({
      name: 'E2E Test Collection',
      slug: 'e2e-test-collection',
      author: authorId,
      schematics: schematicIds,
      description: 'Collection created by Playwright E2E global setup.',
    }),
  });

  if (!createResp.ok) {
    const text = await createResp.text();
    throw new Error(`seedTestCollection failed: ${createResp.status} ${text}`);
  }

  const rec = await createResp.json();
  return { id: rec.id, slug: rec.slug };
}
