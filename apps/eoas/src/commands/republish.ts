import { Env } from '@expo/eas-build-job';
import { Command, Flags } from '@oclif/core';
import ora from 'ora';

import { getAuthExpoHeaders, retrieveExpoCredentials } from '../lib/auth';
import { getExpoConfigUpdateUrl, getPrivateExpoConfigAsync } from '../lib/expoConfig';
import { fetchWithRetries } from '../lib/fetch';
import Log from '../lib/log';
import { isExpoInstalled } from '../lib/package';
import { promptAsync } from '../lib/prompts';
import { resolveVcsClient } from '../lib/vcs';

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
      options: ['ios', 'android'],
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
    const credentials = retrieveExpoCredentials();
    if (!credentials.token && !credentials.sessionSecret) {
      Log.error('You are not logged to eas, please run `eas login`');
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
    let baseUrl: string;
    try {
      const parsedUrl = new URL(updateUrl);
      baseUrl = parsedUrl.origin;
    } catch (e) {
      Log.error('Invalid URL', e);
      process.exit(1);
    }
    const runtimeVersionsEndpoint = `${baseUrl}/api/branch/${branch}/runtimeVersions`;
    const response = await fetchWithRetries(runtimeVersionsEndpoint, {
      headers: { ...getAuthExpoHeaders(credentials), 'use-expo-auth': 'true' },
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
    const updatesEndpoint = `${baseUrl}/api/branch/${branch}/runtimeVersion/${selectedRuntimeVersion.runtimeVersion}/updates`;
    const updatesResponse = await fetchWithRetries(updatesEndpoint, {
      headers: { ...getAuthExpoHeaders(credentials), 'use-expo-auth': 'true' },
    });
    if (!updatesResponse.ok) {
      Log.error(`Failed to fetch updates: ${await updatesResponse.text()}`);
      process.exit(1);
    }
    const updates = (
      (await updatesResponse.json()) as {
        updateUUID: string;
        createdAt: string;
        updateId: string;
        platform: string;
        commitHash: string;
      }[]
    ).filter(u => {
      return u.updateUUID !== 'Rollback to embedded' && u.platform === platform;
    });
    const selectedUpdated = await promptAsync({
      type: 'select',
      name: 'update',
      message: 'Select an update to republish',
      choices: updates.map(update => ({
        title: update.updateUUID,
        value: update,
        description: `Created at: ${update.createdAt}, Platform: ${update.platform}, Commit hash: ${update.commitHash}`,
      })),
    });
    Log.log(`Re-publishing update: ${selectedUpdated.update.updateUUID}`);
    const republishUrl = new URL(`${baseUrl}/republish/${branch}`);
    republishUrl.searchParams.set('platform', platform);
    republishUrl.searchParams.set('runtimeVersion', selectedRuntimeVersion.runtimeVersion);
    republishUrl.searchParams.set('updateId', selectedUpdated.update.updateId);
    republishUrl.searchParams.set('commitHash', selectedUpdated.update.commitHash);
    const republishSpinner = ora('🔄 Republishing update...').start();
    const republishResponse = await fetchWithRetries(republishUrl.toString(), {
      method: 'POST',
      headers: {
        ...getAuthExpoHeaders(credentials),
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
