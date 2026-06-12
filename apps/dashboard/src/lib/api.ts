import { getRefreshToken, getToken, logout, setTokens } from '@/lib/auth.ts';

// All per-app routes (branches, channels, runtime versions, updates,
// updateChannelBranchMapping) are scoped under /api/apps/{appId} on the
// server. The dashboard keeps the currently-selected app id on the ApiClient
// instance so call sites don't all have to pass it explicitly — the
// SelectedAppContext is the single source of truth and calls setAppId()
// whenever the user switches apps.
export class ApiClient {
  private baseUrl: string;
  private appId: string | null = null;

  constructor() {
    // @ts-ignore using window.env for vite
    this.baseUrl = window?.env?.VITE_OTA_API_URL || import.meta.env.VITE_OTA_API_URL;
    if (!this.baseUrl) {
      throw new Error('Missing VITE_OTA_API_URL environment variable');
    }
  }

  public setAppId(appId: string | null) {
    this.appId = appId;
  }

  public getAppId(): string | null {
    return this.appId;
  }

  private appScope(): string {
    if (!this.appId) {
      // Guarded separately from the server 400 so the failure mode is a
      // clear console error instead of a confusing "No app id provided"
      // coming back from the server.
      throw new Error(
        'No app selected — set one via SelectedAppContext before making app-scoped calls.'
      );
    }
    return `/api/apps/${encodeURIComponent(this.appId)}`;
  }

  private populateHeaders(headers: Headers) {
    const token = getToken();
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }
  }
  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const headers = new Headers(options.headers);
    this.populateHeaders(headers);

    const response = await fetch(url, { ...options, headers });
    const refreshToken = getRefreshToken();
    if (response.status === 401 && refreshToken) {
      await this.refreshTokens(refreshToken);
      return this.request<T>(endpoint, options);
    }

    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return response.json() as Promise<T>;
  }

  private async refreshTokens(refreshToken: string) {
    try {
      const form = new URLSearchParams();
      form.append('refreshToken', refreshToken);
      const response = await fetch(`${this.baseUrl}/auth/refreshToken`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: form.toString(),
      });

      if (!response.ok) {
        throw new Error('Failed to refresh token');
      }

      const data = await response.json();
      setTokens(data.token, data.refreshToken);

      localStorage.setItem('accessToken', data.token);
      localStorage.setItem('refreshToken', data.refreshToken);
    } catch (error) {
      console.error('Failed to refresh token:', error);
      logout();
    }
  }

  public async login(password: string) {
    const form = new URLSearchParams();
    form.append('password', password);
    return this.request<{ token: string; refreshToken: string }>(`/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    });
  }
  public async updateChannelBranchMapping(
    branchName: string,
    payload: {
      releaseChannel: string;
    }
  ) {
    return this.request(
      `${this.appScope()}/branch/${encodeURIComponent(branchName)}/updateChannelBranchMapping`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      }
    );
  }
  public async getApps() {
    return this.request<{ id: string; name: string }[]>(`/api/apps`, {
      method: 'GET',
    });
  }
  public async getChannels() {
    return this.request<
      {
        releaseChannelId: string;
        releaseChannelName: string;
        branchName?: string | null;
        branchId?: string | null;
      }[]
    >(`${this.appScope()}/channels`, {
      method: 'GET',
    });
  }
  public async getBranches() {
    return this.request<
      {
        branchName: string;
        branchId: string;
        releaseChannel?: string | null;
      }[]
    >(`${this.appScope()}/branches`, {
      method: 'GET',
    });
  }
  public async getRuntimeVersions(branch: string) {
    return this.request<
      {
        runtimeVersion: string;
        lastUpdatedAt: string;
        createdAt: string;
        numberOfUpdates: number;
      }[]
    >(`${this.appScope()}/branch/${encodeURIComponent(branch)}/runtimeVersions`, {
      method: 'GET',
    });
  }
  public async getUpdates(branch: string, runtimeVersion: string) {
    return this.request<
      {
        updateUUID: string;
        createdAt: string;
        updateId: string;
        platform: string;
        commitHash: string;
        message?: string;
      }[]
    >(
      `${this.appScope()}/branch/${encodeURIComponent(branch)}/runtimeVersion/${encodeURIComponent(runtimeVersion)}/updates`,
      {
        method: 'GET',
      }
    );
  }
  public async getUpdateDetails(branch: string, runtimeVersion: string, updateId: string) {
    return this.request<{
      updateUUID: string;
      createdAt: string;
      updateId: string;
      platform: string;
      commitHash: string;
      message?: string;
      type: number;
      expoConfig: string;
    }>(
      `${this.appScope()}/branch/${encodeURIComponent(branch)}/runtimeVersion/${encodeURIComponent(runtimeVersion)}/updates/${encodeURIComponent(updateId)}`,
      {
        method: 'GET',
      }
    );
  }
  public async getSettings() {
    return this.request<{
      BASE_URL: string;
      CACHE_MODE: string;
      REDIS_HOST: string;
      REDIS_PORT: string;
      STORAGE_MODE: string;
      S3_BUCKET_NAME: string;
      LOCAL_BUCKET_BASE_PATH: string;
      AWS_REGION: string;
      AWS_BASE_ENDPOINT: string;
      AWS_ACCESS_KEY_ID: string;
      CLOUDFRONT_DOMAIN: string;
      CLOUDFRONT_KEY_PAIR_ID: string;
      CLOUDFRONT_PRIVATE_KEY_B64: string;
      AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID: string;
      PRIVATE_LOCAL_CLOUDFRONT_KEY_PATH: string;
      PROMETHEUS_ENABLED: string;
      APPS: { id: string; name?: string }[];
    }>(`/api/settings`, {
      method: 'GET',
    });
  }
}

export const api = new ApiClient();
