import fs from 'fs-extra';
import os from 'os';
import path from 'path';
import { describe, expect, it } from 'vitest';

import { ensurePrivateKeyIgnored, isValidUpdateUrl } from '../utils';

describe('isValidUpdateUrl', () => {
  it('accepts a bare https origin', () => {
    expect(isValidUpdateUrl('https://customota.com')).toBe(true);
    expect(isValidUpdateUrl('http://localhost:3000')).toBe(true);
  });

  it('rejects URLs with a path or without a scheme', () => {
    expect(isValidUpdateUrl('https://customota.com/manifest')).toBe(false);
    expect(isValidUpdateUrl('customota.com')).toBe(false);
    expect(isValidUpdateUrl('ftp://customota.com')).toBe(false);
  });
});

function makeProject(gitignoreContent?: string): string {
  // eslint-disable-next-line node/no-sync
  const projectDir = fs.mkdtempSync(path.join(os.tmpdir(), 'eoas-gitignore-'));
  if (gitignoreContent !== undefined) {
    // eslint-disable-next-line node/no-sync
    fs.writeFileSync(path.join(projectDir, '.gitignore'), gitignoreContent);
  }
  return projectDir;
}

function readGitignore(projectDir: string): string {
  // eslint-disable-next-line node/no-sync
  return fs.readFileSync(path.join(projectDir, '.gitignore'), 'utf8');
}

describe('ensurePrivateKeyIgnored', () => {
  it('creates a .gitignore when the project has none', () => {
    const projectDir = makeProject();
    ensurePrivateKeyIgnored(projectDir);
    expect(readGitignore(projectDir)).toBe(
      '# Code signing private key (server-side secret, never commit it)\nprivate-key.pem\n'
    );
  });

  it('appends to an existing .gitignore without touching its content', () => {
    const projectDir = makeProject('node_modules/\n.expo/\n');
    ensurePrivateKeyIgnored(projectDir);
    const gitignore = readGitignore(projectDir);
    expect(gitignore.startsWith('node_modules/\n.expo/\n')).toBe(true);
    expect(gitignore).toContain('\nprivate-key.pem\n');
  });

  it('terminates the last line when the existing file has no trailing newline', () => {
    const projectDir = makeProject('node_modules/');
    ensurePrivateKeyIgnored(projectDir);
    expect(readGitignore(projectDir)).toContain('node_modules/\n');
    expect(readGitignore(projectDir)).toContain('\nprivate-key.pem\n');
  });

  it('does nothing when any entry already covers the private key', () => {
    const existing = 'certs/private-key.pem\n';
    const projectDir = makeProject(existing);
    ensurePrivateKeyIgnored(projectDir);
    expect(readGitignore(projectDir)).toBe(existing);
  });
});
