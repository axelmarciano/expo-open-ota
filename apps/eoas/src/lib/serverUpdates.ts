import { Credentials, getAuthHeaders } from './auth';
import { fetchWithRetries } from './fetch';

// Read helpers for the server's update listing endpoints, shared by the
// republish and rollback commands (runtime version picker, update picker,
// publish group picker).

export interface RuntimeVersionInfo {
  runtimeVersion: string;
  lastUpdatedAt: string;
  createdAt: string;
  numberOfUpdates: number;
}

export interface ServerUpdateItem {
  updateUUID: string;
  createdAt: string;
  updateId: string;
  platform: string;
  commitHash: string;
  message?: string;
  publishGroup?: string;
}

export async function fetchRuntimeVersions({
  baseUrl,
  appId,
  branch,
  credentials,
}: {
  baseUrl: string;
  appId: string;
  branch: string;
  credentials: Credentials;
}): Promise<RuntimeVersionInfo[]> {
  const response = await fetchWithRetries(
    `${baseUrl}/api/apps/${appId}/branch/${branch}/runtimeVersions`,
    {
      headers: {
        ...getAuthHeaders(credentials),
        'use-cli-auth': 'true',
      },
    }
  );
  if (!response.ok) {
    throw new Error(`Failed to fetch runtime versions: ${await response.text()}`);
  }
  return (await response.json()) as RuntimeVersionInfo[];
}

export async function fetchUpdates({
  baseUrl,
  appId,
  branch,
  runtimeVersion,
  credentials,
}: {
  baseUrl: string;
  appId: string;
  branch: string;
  runtimeVersion: string;
  credentials: Credentials;
}): Promise<ServerUpdateItem[]> {
  const response = await fetchWithRetries(
    `${baseUrl}/api/apps/${appId}/branch/${branch}/runtimeVersion/${runtimeVersion}/updates`,
    {
      headers: {
        ...getAuthHeaders(credentials),
        'use-cli-auth': 'true',
      },
    }
  );
  if (!response.ok) {
    throw new Error(`Failed to fetch updates: ${await response.text()}`);
  }
  return (await response.json()) as ServerUpdateItem[];
}

export interface PublishGroupSummary {
  publishGroup: string;
  platforms: string[];
  commitHash: string;
  message?: string;
  createdAt: string;
  updates: ServerUpdateItem[];
}

// groupPublishedUpdates splits a listing into publish groups (newest first)
// and the leftover ungrouped updates (older CLIs, stateless servers). Filter
// out rollback markers before calling if they should not be offered.
export function groupPublishedUpdates(updates: ServerUpdateItem[]): {
  groups: PublishGroupSummary[];
  ungrouped: ServerUpdateItem[];
} {
  const groupsById = new Map<string, PublishGroupSummary>();
  const ungrouped: ServerUpdateItem[] = [];
  for (const update of updates) {
    if (!update.publishGroup) {
      ungrouped.push(update);
      continue;
    }
    const existing = groupsById.get(update.publishGroup);
    if (!existing) {
      groupsById.set(update.publishGroup, {
        publishGroup: update.publishGroup,
        platforms: [update.platform],
        commitHash: update.commitHash,
        message: update.message,
        createdAt: update.createdAt,
        updates: [update],
      });
      continue;
    }
    existing.updates.push(update);
    if (!existing.platforms.includes(update.platform)) {
      existing.platforms.push(update.platform);
    }
    // The freshest member dates the group in the picker.
    if (update.createdAt > existing.createdAt) {
      existing.createdAt = update.createdAt;
    }
  }
  const groups = [...groupsById.values()].sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  return { groups, ungrouped };
}

// describePublishGroup renders one picker line: the message (or commit) plus
// the platforms it covers.
export function describePublishGroup(group: PublishGroupSummary): {
  title: string;
  description: string;
} {
  // A publish made outside a git repository stores an empty commit hash; fall
  // back to the date so the picker never renders an empty label.
  const shortCommit = group.commitHash.slice(0, 7);
  const label = group.message?.trim()
    ? group.message
    : shortCommit
      ? `Commit ${shortCommit}`
      : `Published ${new Date(group.createdAt).toUTCString()}`;
  const commitSuffix = shortCommit ? `, commit ${shortCommit}` : '';
  return {
    title: `${label} (${group.platforms.join(' + ')})`,
    description: `Published ${new Date(group.createdAt).toUTCString()}${commitSuffix}`,
  };
}
