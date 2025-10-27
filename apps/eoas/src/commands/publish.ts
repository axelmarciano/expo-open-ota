import { Env, Platform } from '@expo/eas-build-job';
import spawnAsync from '@expo/spawn-async';
import { Command, Flags } from '@oclif/core';
import FormData from 'form-data';
import fs from 'fs-extra';
import mime from 'mime';
import path from 'path';

import { RequestUploadUrlItem, computeFilesRequests, requestUploadUrls } from '../lib/assets';
import { getAuthExpoHeaders, retrieveExpoCredentials } from '../lib/auth';
import {
  RequestedPlatform,
  getPrivateExpoConfigAsync,
  getPublicExpoConfigAsync,
  resolveServerUrl,
} from '../lib/expoConfig';
import { fetchWithRetries } from '../lib/fetch';
import Log from '../lib/log';
import { ora } from '../lib/ora';
import { isExpoInstalled } from '../lib/package';
import { confirmAsync } from '../lib/prompts';
import { ensureRepoIsCleanAsync } from '../lib/repo';
import { resolveRuntimeVersionAsync } from '../lib/runtimeVersion';
import { resolveVcsClient } from '../lib/vcs';
import { resolveWorkflowAsync } from '../lib/workflow';

export default class Publish extends Command {
  static override args = {};
  static override description = 'Publish a new update to the self-hosted update server';
  static override examples = ['<%= config.bin %> <%= command.id %>'];
  static override flags = {
    platform: Flags.string({
      type: 'option',
      options: Object.values(RequestedPlatform),
      default: RequestedPlatform.All,
      required: false,
    }),
    channel: Flags.string({
      description: 'Name of the channel to publish the update to',
      required: false,
      deprecated: {
        message:
          'Channel was initially used to provide RELEASE_CHANNEL in the environment when resolving the runtime version. It is no longer needed, you can use RELEASE_CHANNEL={channel} eoas publish --branch={branch} instead',
      },
    }),
    disableRepositoryCheck: Flags.boolean({
      description: 'Disable repository check (Useful for CI/CD)',
      default: false,
      hidden: true,
    }),
    branch: Flags.string({
      description: 'Name of the branch to point to',
      required: true,
    }),
    nonInteractive: Flags.boolean({
      description: 'Run command in non-interactive mode',
      default: false,
    }),
    outputDir: Flags.string({
      description:
        "Where to write build output. You can override the default dist output directory if it's being used by something else",
      default: 'dist',
    }),
  };
  private sanitizeFlags(flags: any): {
    platform: RequestedPlatform;
    branch: string;
    nonInteractive: boolean;
    disableRepositoryCheck: boolean;
    outputDir: string;
    providedDeprecatedChannel?: string;
  } {
    return {
      disableRepositoryCheck: flags.disableRepositoryCheck,
      platform: flags.platform,
      branch: flags.branch,
      nonInteractive: flags.nonInteractive,
      outputDir: flags.outputDir,
      providedDeprecatedChannel: flags.channel,
    };
  }
  public async run(): Promise<void> {
    const credentials = retrieveExpoCredentials();

    if (!credentials.token && !credentials.sessionSecret) {
      Log.error('You are not logged to eas, please run `eas login`');
      process.exit(1);
    }
    const { flags } = await this.parse(Publish);
    const {
      platform,
      nonInteractive,
      branch,
      outputDir,
      providedDeprecatedChannel,
      disableRepositoryCheck,
    } = this.sanitizeFlags(flags);
    if (!branch) {
      Log.error('Branch name is required');
      process.exit(1);
    }
    const projectDir = process.cwd();
    const hasExpo = isExpoInstalled(projectDir);
    if (!hasExpo) {
      Log.error('Expo is not installed in this project. Please install Expo first.');
      process.exit(1);
    }
    const vcsClient = resolveVcsClient(true);
    if (!disableRepositoryCheck) {
      await ensureRepoIsCleanAsync(vcsClient, nonInteractive);
    }
    const config = await getPrivateExpoConfigAsync(projectDir, {
      env: {
        ...(process.env as Env),
        ...(providedDeprecatedChannel ? { RELEASE_CHANNEL: providedDeprecatedChannel } : {}),
      },
    });
    const serverUrl = await resolveServerUrl(config).catch(e => {
      Log.error(e.message);
      process.exit(1);
    });
    if (!nonInteractive) {
      const confirmed = await confirmAsync({
        message: `Is this the correct URL of your self-hosted update server? ${serverUrl}`,
        name: 'export',
        type: 'confirm',
      });
      if (!confirmed) {
        Log.error('Please run `eoas init` to setup the correct update url');
        process.exit(1);
      }
    }

    const commitHash = await vcsClient.getCommitHashAsync();

    const runtimeSpinner = ora('🔄 Resolving runtime version...').start();
    const runtimeVersions = [
      ...(!platform || platform === RequestedPlatform.All || platform === RequestedPlatform.Ios
        ? [
            {
              runtimeVersion: (
                await resolveRuntimeVersionAsync({
                  exp: config,
                  platform: 'ios',
                  workflow: await resolveWorkflowAsync(projectDir, Platform.IOS, vcsClient),
                  projectDir,
                  env: {
                    ...(process.env as Env),
                    ...(providedDeprecatedChannel
                      ? { RELEASE_CHANNEL: providedDeprecatedChannel }
                      : {}),
                  },
                })
              )?.runtimeVersion,
              platform: 'ios',
            },
          ]
        : []),
      ...(!platform || platform === RequestedPlatform.All || platform === RequestedPlatform.Android
        ? [
            {
              runtimeVersion: (
                await resolveRuntimeVersionAsync({
                  exp: config,
                  platform: 'android',
                  workflow: await resolveWorkflowAsync(projectDir, Platform.ANDROID, vcsClient),
                  projectDir,
                  env: {
                    ...(process.env as Env),
                    ...(providedDeprecatedChannel
                      ? { RELEASE_CHANNEL: providedDeprecatedChannel }
                      : {}),
                  },
                })
              )?.runtimeVersion,
              platform: 'android',
            },
          ]
        : []),
    ].filter(({ runtimeVersion }) => !!runtimeVersion);
    if (!runtimeVersions.length) {
      runtimeSpinner.fail('Could not resolve runtime versions for the requested platforms');
      Log.error('Could not resolve runtime versions for the requested platforms');
      process.exit(1);
    }
    runtimeSpinner.succeed('✅ Runtime versions resolved');
    const cleaningSpinner = ora(`🗑️ Cleaning up ${outputDir} directory...`).start();
    try {
      await fs.remove(path.join(projectDir, outputDir));
      cleaningSpinner.succeed('✅ Cleanup completed');
    } catch (e) {
      cleaningSpinner.fail('❌ Failed to clean up the output directory');
      Log.error(e);
      process.exit(1);
    }
    const exportSpinner = ora('📦 Exporting project files...').start();
    try {
      const specifiedPlatform = platform === RequestedPlatform.All ? [] : ['--platform', platform];
      const { stdout } = await spawnAsync('npx', ['expo', 'export', '--output-dir', outputDir, ...specifiedPlatform], {
        cwd: projectDir,
        env: {
          ...process.env,
          EXPO_NO_DOTENV: '1',
        },
      });
      exportSpinner.succeed('🚀 Project exported successfully');
      Log.withInfo(stdout);
    } catch (e) {
      exportSpinner.fail(`❌ Failed to export the project, ${e}`);
      process.exit(1);
    }
    const publicConfig = await getPublicExpoConfigAsync(projectDir, {
      skipSDKVersionRequirement: true,
    });
    if (!publicConfig) {
      Log.error(
        'Could not find Expo config in this project. Please make sure you have an Expo config.'
      );
      process.exit(1);
    }
    // eslint-disable-next-line
    fs.writeJsonSync(path.join(projectDir, outputDir, 'expoConfig.json'), publicConfig, {
      spaces: 2,
    });
    Log.withInfo(`expoConfig.json file created in ${outputDir} directory`);
    const uploadFilesSpinner = ora('📤 Uploading files...').start();
    const files = computeFilesRequests(projectDir, outputDir, platform || RequestedPlatform.All);
    if (!files.length) {
      uploadFilesSpinner.fail('No files to upload');
      process.exit(1);
    }
    let uploadUrls: {
      uploadRequests: RequestUploadUrlItem[];
      updateId: string;
      platform: string;
      runtimeVersion: string;
    }[] = [];
    try {
      uploadUrls = await Promise.all(
        runtimeVersions.map(async ({ runtimeVersion, platform }) => {
          if (!runtimeVersion) {
            throw new Error('Runtime version is not resolved');
          }
          return {
            ...(await requestUploadUrls({
              body: {
                fileNames: files.map(file => file.path),
              },
              requestUploadUrl: `${serverUrl}/requestUploadUrl/${branch}`,
              auth: credentials,
              runtimeVersion,
              platform,
              commitHash,
            })),
            runtimeVersion,
            platform,
          };
        })
      );
      const allItems = uploadUrls.flatMap(({ uploadRequests }) => uploadRequests);
      await Promise.all(
        allItems.map(async itm => {
          const isLocalBucketFileUpload = itm.requestUploadUrl.startsWith(
            `${serverUrl}/uploadLocalFile`
          );
          const formData = new FormData();
          let file: fs.ReadStream;
          try {
            file = fs.createReadStream(path.join(projectDir, outputDir, itm.filePath));
          } catch {
            throw new Error(`Failed to read file ${itm.filePath}`);
          }
          formData.append(itm.fileName, file);
          if (isLocalBucketFileUpload) {
            const response = await fetchWithRetries(itm.requestUploadUrl, {
              method: 'PUT',
              headers: {
                ...formData.getHeaders(),
                ...getAuthExpoHeaders(credentials),
              },
              body: formData,
            });
            if (!response.ok) {
              Log.error('Failed to upload file', await response.text());
              throw new Error('Failed to upload file');
            }
            file.close();
            return;
          }
          const findFile = files.find(f => f.path === itm.filePath || f.name === itm.fileName);
          if (!findFile) {
            Log.error(`File ${itm.filePath} not found`);
            throw new Error(`File ${itm.filePath} not found`);
          }
          let contentType = mime.getType(findFile.ext);
          if (!contentType) {
            contentType = 'application/octet-stream';
          }
          const buffer = await fs.readFile(path.join(projectDir, outputDir, itm.filePath));
          const response = await fetchWithRetries(itm.requestUploadUrl, {
            method: 'PUT',
            headers: {
              'Content-Type': contentType,
              'Cache-Control': 'max-age=31556926',
            },
            body: buffer,
          });
          if (!response.ok) {
            Log.error('❌ File upload failed', await response.text());
            process.exit(1);
          }
          file.close();
        })
      );
      uploadFilesSpinner.succeed('✅ Files uploaded successfully');
    } catch (e) {
      uploadFilesSpinner.fail('❌ Failed to upload static files');
      Log.error(e);
      process.exit(1);
    }

    const markAsFinishedSpinner = ora('🔗 Marking the updates as finished...').start();
    const results = await Promise.all(
      uploadUrls.map(async ({ updateId, platform, runtimeVersion }) => {
        const response = await fetchWithRetries(
          `${serverUrl}/markUpdateAsUploaded/${branch}?platform=${platform}&updateId=${updateId}&runtimeVersion=${runtimeVersion}`,
          {
            method: 'POST',
            headers: {
              ...getAuthExpoHeaders(credentials),
              'Content-Type': 'application/json',
            },
          }
        );
        // If success and status code = 200
        if (response.ok) {
          Log.withInfo(`✅ Update ready for ${platform}`);
          return 'deployed';
        }
        // If response.status === 406 duplicate update
        if (response.status === 406) {
          Log.withInfo(`⚠️ There is no change in the update for ${platform}, ignored...`);
          return 'identical';
        }
        Log.error('❌ Failed to mark the update as finished for platform', platform);
        Log.newLine();
        Log.error(await response.text());
        return 'error';
      })
    );
    const erroredUpdates = results.filter(result => result === 'error');
    const hasSuccess = results.some(result => result === 'deployed');
    const allIdentical = results.every(result => result === 'identical');
    if (allIdentical) {
      markAsFinishedSpinner.warn('⚠️ No changes found in the update, nothing to deploy');
      return;
    }
    if (erroredUpdates.length) {
      markAsFinishedSpinner.fail('❌ Some errors occurred while marking updates as finished');
      throw new Error();
    } else {
      markAsFinishedSpinner.succeed(
        `\n✅ Your update has been successfully pushed to ${serverUrl}`
      );
    }
    if (hasSuccess) {
      Log.withInfo(`🌿 Branch: \`${branch}\``);
      Log.withInfo(`⏳ Deployed at: \`${new Date().toUTCString()}\`\n`);
      Log.withInfo('🔥 Your users will receive the latest update automatically!');
    }
  }
}
