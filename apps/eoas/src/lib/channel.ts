import { getAuthHeaders } from './auth';
import { fetchWithRetries } from './fetch';

export async function resolveReleaseChannelDynamicallyFromBranch(
  baseUrl: string,
  branch: string
): Promise<string> {
  const branchesEndpoint = `${baseUrl}/api/branches`;
  const response = await fetchWithRetries(branchesEndpoint, {
    headers: { ...getAuthHeaders() },
  });
  if (!response.ok) {
    throw new Error(`Failed to retrieve branches from server: ${await response.text()}`);
  }
  const branches = (await response.json()) as {
    branchName: string;
    releaseChannel?: string;
  }[];
  const branchInfo = branches.find(b => b.branchName === branch);
  if (!branchInfo) {
    throw new Error(`Branch ${branch} not found`);
  }
  if (!branchInfo.releaseChannel) {
    throw new Error(`Branch ${branch} does not have a release channel linked`);
  }
  return branchInfo.releaseChannel;
}
