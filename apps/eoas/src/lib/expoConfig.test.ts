import assert from 'node:assert/strict';
import test from 'node:test';

import { preserveSchemesInPublicExpoConfig } from './expoConfig';

void test('preserves missing top-level and platform schemes from the private config', () => {
  const normalized = preserveSchemesInPublicExpoConfig(
    {
      name: 'demo',
      slug: 'demo',
      version: '1.0.0',
      ios: {
        bundleIdentifier: 'com.example.demo',
      },
      android: {
        package: 'com.example.demo',
      },
    },
    {
      name: 'demo',
      slug: 'demo',
      version: '1.0.0',
      scheme: 'demo',
      ios: {
        bundleIdentifier: 'com.example.demo',
        scheme: 'demo-ios',
      },
      android: {
        package: 'com.example.demo',
        scheme: 'demo-android',
      },
    } as any
  );

  assert.equal(normalized.scheme, 'demo');
  assert.equal(normalized.ios?.scheme, 'demo-ios');
  assert.equal(normalized.android?.scheme, 'demo-android');
});

void test('backfills platform schemes from the normalized top-level scheme when needed', () => {
  const normalized = preserveSchemesInPublicExpoConfig(
    {
      name: 'demo',
      slug: 'demo',
      version: '1.0.0',
      scheme: 'demo',
    },
    {
      name: 'demo',
      slug: 'demo',
      version: '1.0.0',
      scheme: 'demo',
    } as any
  );

  assert.equal(normalized.scheme, 'demo');
  assert.equal(normalized.ios?.scheme, 'demo');
  assert.equal(normalized.android?.scheme, 'demo');
});

void test('keeps public scheme values when they are already present', () => {
  const normalized = preserveSchemesInPublicExpoConfig(
    {
      name: 'demo',
      slug: 'demo',
      version: '1.0.0',
      scheme: 'public-scheme',
      ios: {
        bundleIdentifier: 'com.example.demo',
        scheme: 'public-ios',
      },
      android: {
        package: 'com.example.demo',
        scheme: 'public-android',
      },
    },
    {
      name: 'demo',
      slug: 'demo',
      version: '1.0.0',
      scheme: 'private-scheme',
      ios: {
        bundleIdentifier: 'com.example.demo',
        scheme: 'private-ios',
      },
      android: {
        package: 'com.example.demo',
        scheme: 'private-android',
      },
    } as any
  );

  assert.equal(normalized.scheme, 'public-scheme');
  assert.equal(normalized.ios?.scheme, 'public-ios');
  assert.equal(normalized.android?.scheme, 'public-android');
});
