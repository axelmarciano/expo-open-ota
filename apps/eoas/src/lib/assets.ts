// This file is partially copied from eas-cli[https://github.com/expo/eas-cli] to ensure consistent user experience across the CLI.
import { Platform } from '@expo/config';
import fs from 'fs-extra';
import Joi from 'joi';
import path from 'path';

import { Credentials, getAuthHeaders } from './auth';
import { RequestedPlatform } from './expoConfig';
import { fetchWithRetries } from './fetch';
import Log from './log';

const fileMetadataJoi = Joi.object({
  assets: Joi.array()
    .required()
    .items(Joi.object({ path: Joi.string().required(), ext: Joi.string().required() })),
  bundle: Joi.string().required(),
}).optional();
export const MetadataJoi = Joi.object({
  version: Joi.number().required(),
  bundler: Joi.string().required(),
  fileMetadata: Joi.object({
    android: fileMetadataJoi,
    ios: fileMetadataJoi,
    web: fileMetadataJoi,
  }).required(),
}).required();

type Metadata = {
  version: number;
  bundler: 'metro';
  fileMetadata: {
    [key in Platform]: { assets: { path: string; ext: string }[]; bundle: string };
  };
};

interface AssetToUpload {
  path: string;
  name: string;
  ext: string;
}

function loadMetadata(distRoot: string): Metadata {
  // eslint-disable-next-line
  const fileContent = fs.readFileSync(path.join(distRoot, 'metadata.json'), 'utf8');
  let metadata: Metadata;
  try {
    metadata = JSON.parse(fileContent);
  } catch (e: any) {
    Log.error(`Failed to read metadata.json: ${e.message}`);
    throw e;
  }
  const { error } = MetadataJoi.validate(metadata);
  if (error) {
    throw error;
  }
  // Check version and bundler by hand (instead of with Joi) so
  // more informative error messages can be returned.
  if (metadata.version !== 0) {
    throw new Error('Only bundles with metadata version 0 are supported');
  }
  if (metadata.bundler !== 'metro') {
    throw new Error('Only bundles created with Metro are currently supported');
  }
  const platforms = Object.keys(metadata.fileMetadata);
  if (platforms.length === 0) {
    Log.warn('No updates were exported for any platform');
  }
  Log.debug(`Loaded ${platforms.length} platform(s): ${platforms.join(', ')}`);
  return metadata;
}

export function computeFilesRequests(
  projectDir: string,
  outputDir: string,
  requestedPlatform: RequestedPlatform
): AssetToUpload[] {
  const metadata = loadMetadata(path.join(projectDir, outputDir));
  const assets: AssetToUpload[] = [
    { path: 'metadata.json', name: 'metadata.json', ext: 'json' },
    { path: 'expoConfig.json', name: 'expoConfig.json', ext: 'json' },
  ];
  for (const platform of Object.keys(metadata.fileMetadata) as Platform[]) {
    if (requestedPlatform !== RequestedPlatform.All && requestedPlatform !== platform) {
      continue;
    }
    const bundle = metadata.fileMetadata[platform].bundle;
    assets.push({ path: bundle, name: path.basename(bundle), ext: 'hbc' });
    for (const asset of metadata.fileMetadata[platform].assets) {
      assets.push({ path: asset.path, name: path.basename(asset.path), ext: asset.ext });
    }
  }
  return assets;
}

export interface RequestUploadUrlItem {
  requestUploadUrl: string;
  fileName: string;
  filePath: string;
  // Extra headers the server requires on the PUT to requestUploadUrl
  // (e.g. x-ms-blob-type for Azure Blob Storage). Absent on older servers.
  headers?: Record<string, string>;
}

export function activeRolloutConflictMessage(branch: string): string {
  return `A progressive rollout is already active for branch "${branch}" on this runtime version. End or revert it from the dashboard before publishing a new update.`;
}

export async function requestUploadUrls({
  body,
  requestUploadUrl,
  auth,
  runtimeVersion,
  platform,
  commitHash,
  message,
  rolloutPercentage,
  branch,
}: {
  body: { fileNames: string[] };
  requestUploadUrl: string;
  auth: Credentials;
  runtimeVersion: string;
  platform: string;
  commitHash?: string;
  message?: string;
  rolloutPercentage?: number;
  branch: string;
}): Promise<{
  uploadRequests: RequestUploadUrlItem[];
  updateId: string;
  rolloutPercentage?: number;
}> {
  const uploadUrl = new URL(requestUploadUrl);
  uploadUrl.searchParams.set('runtimeVersion', runtimeVersion);
  uploadUrl.searchParams.set('platform', platform);
  uploadUrl.searchParams.set('commitHash', commitHash ?? '');
  if (rolloutPercentage !== undefined) {
    uploadUrl.searchParams.set('rolloutPercentage', String(rolloutPercentage));
  }

  const requestBody: { fileNames: string[]; message?: string } = { ...body };
  if (message) {
    requestBody.message = message;
  }

  const response = await fetchWithRetries(uploadUrl.toString(), {
    method: 'POST',
    headers: {
      ...getAuthHeaders(auth),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(requestBody),
  });
  if (response.status === 409) {
    throw new Error(activeRolloutConflictMessage(branch));
  }
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Failed to request upload URL: ${text}`);
  }
  const json = await response.json();
  // An old server silently ignores unknown query params, so a missing echo means
  // the rollout was not applied even though the flag was set. Abort before any
  // file is uploaded: continuing would finalize a full 100% publish.
  if (rolloutPercentage !== undefined && json.rolloutPercentage === undefined) {
    throw new Error(
      'The server ignored --rollout-percentage and would publish to 100% of devices. Update the server to a version that supports progressive rollouts, or publish without --rollout-percentage.'
    );
  }
  return json;
}
