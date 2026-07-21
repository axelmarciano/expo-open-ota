import fs from 'fs-extra';
import path from 'path';

import Log from './log';

export function isValidUpdateUrl(updateUrl: string): boolean {
  return updateUrl.match(/^https?:\/\/[^/]+$/) !== null;
}

// Keeps the code signing private key out of the app repository: appends a
// 'private-key.pem' pattern to the project .gitignore unless any entry already
// mentions it (a pattern without a slash matches at every directory level).
export function ensurePrivateKeyIgnored(projectDir: string): void {
  const gitignorePath = path.join(projectDir, '.gitignore');
  try {
    // eslint-disable-next-line node/no-sync
    const gitignore = fs.existsSync(gitignorePath) ? fs.readFileSync(gitignorePath, 'utf8') : '';
    if (gitignore.includes('private-key.pem')) {
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
