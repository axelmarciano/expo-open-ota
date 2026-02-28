import { Env } from '@expo/eas-build-job';
import { Command, Flags } from '@oclif/core';

import { getAuthHeaders } from '../lib/auth';
import { getExpoConfigUpdateUrl, getPrivateExpoConfigAsync } from '../lib/expoConfig';
import { fetchWithRetries } from '../lib/fetch';
import Log from '../lib/log';
import { isExpoInstalled } from '../lib/package';

export default class SetChannelBranch extends Command {
  static override args = {};
  static override description = 'Update the branch mapping of an existing channel';
  static override examples = [
    '<%= config.bin %> <%= command.id %> --channel production --branch main',
  ];
  static override flags = {
    channel: Flags.string({
      description: 'Name of the channel to update',
      required: true,
    }),
    branch: Flags.string({
      description: 'Name of the branch to map the channel to',
      required: true,
    }),
  };

  public async run(): Promise<void> {
    const { flags } = await this.parse(SetChannelBranch);
    const { channel, branch } = flags;

    const projectDir = process.cwd();
    if (!isExpoInstalled(projectDir)) {
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

    const endpoint = `${baseUrl}/api/branch/${encodeURIComponent(
      branch
    )}/updateChannelBranchMapping`;
    const response = await fetchWithRetries(endpoint, {
      method: 'POST',
      headers: {
        ...getAuthHeaders(),
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ releaseChannel: channel }),
    });

    if (!response.ok) {
      Log.error(`Failed to update channel branch mapping: ${await response.text()}`);
      process.exit(1);
    }

    Log.withInfo(`Channel "${channel}" is now mapped to branch "${branch}"`);
  }
}
