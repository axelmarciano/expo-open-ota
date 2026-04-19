// This file is partially copied from eas-cli[https://github.com/expo/eas-cli] to ensure consistent user experience across the CLI.
import { Platform } from '@expo/config';
import fs from 'fs-extra';
import Joi from 'joi';
import path from 'path';

import { ExpoCredentials, getAuthExpoHeaders } from './auth';
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
}

export async function requestUploadUrls({
  body,
  requestUploadUrl,
  auth,
  runtimeVersion,
  platform,
  commitHash,
  message,
}: {
  body: { fileNames: string[] };
  requestUploadUrl: string;
  auth: ExpoCredentials;
  runtimeVersion: string;
  platform: string;
  commitHash?: string;
  message?: string;
}): Promise<{ uploadRequests: RequestUploadUrlItem[]; updateId: string }> {
  const uploadUrl = new URL(requestUploadUrl);
  uploadUrl.searchParams.set('runtimeVersion', runtimeVersion);
  uploadUrl.searchParams.set('platform', platform);
  uploadUrl.searchParams.set('commitHash', commitHash ?? '');

  const requestBody: { fileNames: string[]; message?: string } = { ...body };
  if (message) {
    requestBody.message = message;
  }

  const response = await fetchWithRetries(uploadUrl.toString(), {
    method: 'POST',
    headers: {
      ...getAuthExpoHeaders(auth),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(requestBody),
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Failed to request upload URL: ${text}`);
  }
  return await response.json();
}
