// Computes the integrity bundle hash (sha384 of the built remoteEntry) and writes it into the
// manifest's integrity.bundleHash. Runs after `ng build` as the second half of `pnpm build`. The
// hash is the SRI the shell loads the remote with, so it must match the served bundle byte-for-byte.
// Changing the bundle invalidates any prior signature, so the signature is cleared here and the
// manifest must be re-signed (`flyspace-ext sign`) before publishing.
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';

const ENTRY = 'dist/template/browser/remoteEntry.json';
const MANIFEST = 'flyspace-extension.json';

const hash = 'sha384-' + createHash('sha384').update(readFileSync(ENTRY)).digest('base64');

const manifest = JSON.parse(readFileSync(MANIFEST, 'utf8'));
manifest.integrity = manifest.integrity ?? {};
manifest.integrity.bundleHash = hash;
delete manifest.integrity.signature;

writeFileSync(MANIFEST, JSON.stringify(manifest, null, 2) + '\n');
console.log(`bundleHash ${hash}`);
