import { describe, expect, it, vi } from 'vitest';

import { activeRolloutConflictMessage, requestUploadUrls } from '../assets';
import { fetchWithRetries } from '../fetch';

vi.mock('../fetch', () => ({
  fetchWithRetries: vi.fn(),
}));

const credentials = { token: 'test-token', sessionSecret: undefined };

function baseParams(): Parameters<typeof requestUploadUrls>[0] {
  return {
    body: { fileNames: ['bundle.js'] },
    requestUploadUrl: 'https://ota.example.com/app-1/requestUploadUrl/main',
    auth: credentials,
    runtimeVersion: '1.0.0',
    platform: 'ios',
    commitHash: 'abc1234',
    branch: 'main',
  };
}

function respondWith(payload: unknown): void {
  vi.mocked(fetchWithRetries).mockResolvedValueOnce({
    ok: true,
    status: 200,
    json: async () => payload,
  } as Response);
}

function requestedUrl(): URL {
  const calls = vi.mocked(fetchWithRetries).mock.calls;
  return new URL(String(calls[calls.length - 1][0]));
}

describe('requestUploadUrls publish group wire contract', () => {
  it('sends the publish group as a query parameter and returns the acknowledgment', async () => {
    respondWith({ updateId: '1', uploadRequests: [], publishGroup: 'group-a' });
    const result = await requestUploadUrls({ ...baseParams(), publishGroup: 'group-a' });
    expect(requestedUrl().searchParams.get('publishGroup')).toBe('group-a');
    expect(result.publishGroup).toBe('group-a');
  });

  it('omits the parameter entirely when no group is provided', async () => {
    respondWith({ updateId: '1', uploadRequests: [] });
    const result = await requestUploadUrls(baseParams());
    expect(requestedUrl().searchParams.has('publishGroup')).toBe(false);
    expect(result.publishGroup).toBeUndefined();
  });

  it('leaves the acknowledgment undefined when the server ignores the group', async () => {
    // Old server or stateless mode: no echo. The publish command reads this
    // to print the ungrouped note instead of a publish group line.
    respondWith({ updateId: '1', uploadRequests: [] });
    const result = await requestUploadUrls({ ...baseParams(), publishGroup: 'group-a' });
    expect(result.publishGroup).toBeUndefined();
  });
});

describe('requestUploadUrls rollout guardrails', () => {
  it('aborts when the server does not echo the rollout percentage', async () => {
    respondWith({ updateId: '1', uploadRequests: [] });
    await expect(requestUploadUrls({ ...baseParams(), rolloutPercentage: 10 })).rejects.toThrow(
      /ignored --rollout-percentage/
    );
  });

  it('continues when the rollout percentage is echoed', async () => {
    respondWith({ updateId: '1', uploadRequests: [], rolloutPercentage: 10 });
    const result = await requestUploadUrls({ ...baseParams(), rolloutPercentage: 10 });
    expect(result.rolloutPercentage).toBe(10);
  });

  it('surfaces an active rollout conflict as the dedicated message', async () => {
    vi.mocked(fetchWithRetries).mockResolvedValueOnce({
      ok: false,
      status: 409,
      text: async () => 'conflict',
    } as Response);
    await expect(requestUploadUrls(baseParams())).rejects.toThrow(
      activeRolloutConflictMessage('main')
    );
  });
});
