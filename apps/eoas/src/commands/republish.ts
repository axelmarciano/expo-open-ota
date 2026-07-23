import { Env } from '@expo/eas-build-job';
import { Command, Flags } from '@oclif/core';
import ora from 'ora';

import { getAuthHeaders, retrieveCredentials, validateCredentials } from '../lib/auth';
import {
  getExpoConfigUpdateUrl,
  getPrivateExpoConfigAsync,
  requireExpoAppId,
} from '../lib/expoConfig';
import { fetchWithRetries } from '../lib/fetch';
import Log from '../lib/log';
import { isExpoInstalled } from '../lib/package';
import { promptAsync } from '../lib/prompts';
import { resolveVcsClient } from '../lib/vcs';

type RepublishableUpdate = {
  updateUUID: string;
  createdAt: string;
  updateId: string;
  platform: string;
  commitHash: string;
};

type UpdatesPage = {
  items: RepublishableUpdate[];
  nextCursor: string | null;
};

type UpdateSelection = { kind: 'update'; update: RepublishableUpdate } | { kind: 'loadMore' };

const UPDATES_PAGE_SIZE = 20;

export default class Publish extends Command {
  static override args = {};
  static override description = 'Republish a previous update to a branch';
  static override examples = ['<%= config.bin %> <%= command.id %>'];
  static override flags = {
    branch: Flags.string({
      description: 'Name of the branch to point to',
      required: true,
    }),
    platform: Flags.string({
      type: 'option',
      options: ['ios', 'android', 'all'],
      default: 'all',
      required: true,
    }),
  };
  private sanitizeFlags(flags: any): {
    branch: string;
    platform: string;
  } {
    return {
      branch: flags.branch,
      platform: flags.platform,
    };
  }
  public async run(): Promise<void> {
    const credentials = retrieveCredentials();
    if (!validateCredentials(credentials)) {
      Log.error(
        'Invalid credentials. Please run `eas login or set EXPO_ACCESS_TOKEN or EOO_TOKEN environment variable`'
      );
      process.exit(1);
    }
    const { flags } = await this.parse(Publish);
    const { branch, platform } = this.sanitizeFlags(flags);
    if (!branch) {
      Log.error('Branch name is required');
      process.exit(1);
    }
    if (!platform) {
      Log.error('Platform is required');
      process.exit(1);
    }
    const vcsClient = resolveVcsClient(true);
    await vcsClient.ensureRepoExistsAsync();
    // const commitHash = await vcsClient.getCommitHashAsync();
    const projectDir = process.cwd();
    const hasExpo = isExpoInstalled(projectDir);
    if (!hasExpo) {
      Log.error('Expo is not installed in this project. Please install Expo first.');
      process.exit(1);
    }
    const privateConfig = await getPrivateExpoConfigAsync(projectDir, {
      env: process.env as Env,
    });
    const updateUrl = getExpoConfigUpdateUrl(privateConfig);
    if (!updateUrl) {
      Log.error(
        "Update url is not setup in your config. Please run 'eoas init' to setup the update url"
      );
      process.exit(1);
    }
    const appId = requireExpoAppId(privateConfig);
    let baseUrl: string;
    try {
      const parsedUrl = new URL(updateUrl);
      baseUrl = parsedUrl.origin;
    } catch (e) {
      Log.error('Invalid URL', e);
      process.exit(1);
    }
    const runtimeVersionsEndpoint = `${baseUrl}/api/apps/${appId}/branch/${branch}/runtimeVersions`;
    const response = await fetchWithRetries(runtimeVersionsEndpoint, {
      headers: {
        ...getAuthHeaders(credentials),
        'use-cli-auth': 'true',
      },
    });
    if (!response.ok) {
      Log.error(`Failed to fetch runtime versions: ${await response.text()}`);
      process.exit(1);
    }
    const runtimeVersions = (await response.json()) as {
      runtimeVersion: string;
      lastUpdatedAt: string;
      createdAt: string;
      numberOfUpdates: number;
    }[];
    const filteredRuntimeVersions = runtimeVersions.filter(
      runtimeVersion => runtimeVersion.numberOfUpdates > 1
    );
    if (filteredRuntimeVersions.length === 0) {
      Log.error('No runtime versions found');
      process.exit(1);
    }
    // Ask the user to select a runtime version
    const selectedRuntimeVersion = await promptAsync({
      type: 'select',
      name: 'runtimeVersion',
      message: 'Select a runtime version',
      choices: filteredRuntimeVersions.map(runtimeVersion => ({
        title: runtimeVersion.runtimeVersion,
        value: runtimeVersion.runtimeVersion,
      })),
    });
    Log.log(`Selected runtime version: ${selectedRuntimeVersion.runtimeVersion}`);
    const updatesEndpoint = `${baseUrl}/api/apps/${appId}/branch/${encodeURIComponent(
      branch
    )}/runtimeVersion/${encodeURIComponent(selectedRuntimeVersion.runtimeVersion)}/updates`;
    const updates: RepublishableUpdate[] = [];
    let nextCursor: string | null | undefined;
    const loadNextPage = async (): Promise<void> => {
      const previousCount = updates.length;
      do {
        const url = new URL(updatesEndpoint);
        url.searchParams.set('limit', String(UPDATES_PAGE_SIZE));
        if (nextCursor) {
          url.searchParams.set('cursor', nextCursor);
        }
        const updatesResponse = await fetchWithRetries(url.toString(), {
          headers: {
            ...getAuthHeaders(credentials),
            'use-cli-auth': 'true',
          },
        });
        if (!updatesResponse.ok) {
          Log.error(`Failed to fetch updates: ${await updatesResponse.text()}`);
          process.exit(1);
        }
        const page = (await updatesResponse.json()) as UpdatesPage;
        updates.push(
          ...page.items.filter(
            update =>
              update.updateUUID !== 'Rollback to embedded' &&
              (platform === 'all' || update.platform === platform)
          )
        );
        nextCursor = page.nextCursor;
      } while (updates.length === previousCount && nextCursor);
    };

    await loadNextPage();
    let selectedUpdate: RepublishableUpdate | undefined;
    let initialChoiceIndex = 0;
    while (!selectedUpdate) {
      if (updates.length === 0 && !nextCursor) {
        Log.error(
          `No republishable updates found for runtime version ${selectedRuntimeVersion.runtimeVersion} on platform ${platform}.`
        );
        process.exit(1);
      }
      const choices: { title: string; value: UpdateSelection; description?: string }[] =
        updates.map(update => ({
          title: update.updateUUID,
          value: { kind: 'update', update },
          description: `Created at: ${update.createdAt}, Platform: ${update.platform}, Commit hash: ${update.commitHash}`,
        }));
      if (nextCursor) {
        choices.push({
          title: 'Load more updates',
          value: { kind: 'loadMore' },
        });
      }
      const answer = await promptAsync({
        type: 'select',
        name: 'selection',
        message: 'Select an update to republish',
        choices,
        initial: initialChoiceIndex,
      });
      const selection = answer.selection as UpdateSelection;
      if (selection.kind === 'loadMore') {
        const firstNewUpdateIndex = updates.length;
        await loadNextPage();
        initialChoiceIndex = Math.min(firstNewUpdateIndex, Math.max(0, updates.length - 1));
      } else {
        selectedUpdate = selection.update;
      }
    }
    Log.log(`Re-publishing update: ${selectedUpdate.updateUUID}`);
    const republishUrl = new URL(`${baseUrl}/${appId}/republish/${branch}`);
    republishUrl.searchParams.set('platform', selectedUpdate.platform);
    republishUrl.searchParams.set('runtimeVersion', selectedRuntimeVersion.runtimeVersion);
    republishUrl.searchParams.set('updateId', selectedUpdate.updateId);
    republishUrl.searchParams.set('commitHash', selectedUpdate.commitHash);
    const republishSpinner = ora('🔄 Republishing update...').start();
    const republishResponse = await fetchWithRetries(republishUrl.toString(), {
      method: 'POST',
      headers: {
        ...getAuthHeaders(credentials),
        'use-cli-auth': 'true',
        'Content-Type': 'application/json',
      },
    });
    if (!republishResponse.ok) {
      republishSpinner.fail('❌ Republish failed');
      Log.error(`Failed to republish update: ${await republishResponse.text()}`);
      process.exit(1);
    }
    republishSpinner.succeed('✅ Republish successful');
  }
}
