import fs from 'fs-extra';
import path from 'path';

import Log from './log';

export function isValidUpdateUrl(updateUrl: string): boolean {
  return updateUrl.match(/^https?:\/\/[^/]+$/) !== null;
}

// Keeps the code signing private key out of the app repository: appends a bare
// 'private-key.pem' pattern to the project .gitignore (a pattern without a
// slash matches at every directory level). Only an existing bare rule counts:
// comments, negated entries and path-specific rules like certs/private-key.pem
// do not guarantee project-wide protection. Appending at the end also wins over
// an earlier negated entry, since the last matching gitignore rule prevails.
export function ensurePrivateKeyIgnored(projectDir: string): void {
  const gitignorePath = path.join(projectDir, '.gitignore');
  try {
    // eslint-disable-next-line node/no-sync
    const gitignore = fs.existsSync(gitignorePath) ? fs.readFileSync(gitignorePath, 'utf8') : '';
    const lines = gitignore.split(/\r?\n/).map(line => line.trim());
    const lastBareRule = lines.lastIndexOf('private-key.pem');
    const lastNegation = lines.lastIndexOf('!private-key.pem');
    if (lastBareRule !== -1 && lastBareRule > lastNegation) {
      return;
    }
    const separator = gitignore === '' ? '' : gitignore.endsWith('\n') ? '\n' : '\n\n';
    // eslint-disable-next-line node/no-sync
    fs.appendFileSync(
      gitignorePath,
      `${separator}# Code signing private key (server-side secret, never commit it)\nprivate-key.pem\n`
    );
    Log.succeed('Added private-key.pem to .gitignore');
  } catch {
    Log.warn(
      'Could not update .gitignore. Make sure private-key.pem is never committed to your repository.'
    );
  }
}
