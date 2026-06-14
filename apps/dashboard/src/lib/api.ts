import { getRefreshToken, getToken, logout, setTokens } from '@/lib/auth.ts';

export type APIProblemPayload = {
  title: string;
  status: number;
  detail: string;
};

export class ApiProblemError extends Error {
  public title: string;
  public status: number;
  public detail: string;

  constructor(payload: APIProblemPayload) {
    super(payload.detail);
    this.name = 'ApiProblemError';
    this.title = payload.title;
    this.status = payload.status;
    this.detail = payload.detail;
  }
}

export type KeysMode = 'local' | 'aws-secrets-manager' | 'environment' | 'database';

export type KeysConfig = {
  mode: KeysMode;
  publicPath?: string;
  privatePath?: string;
  publicSecretId?: string;
  privateSecretId?: string;
  publicB64?: string;
  privateB64?: string;
  sealedPublicKey?: string;
  sealedPrivateKey?: string;
};

export type AppDescriptor = {
  id: string;
  name?: string;
};

export type AppDetails = AppDescriptor & {
  keys: KeysConfig;
  createdAt?: number;
};

export type CreateAppPayload = {
  name: string;
  keysConfig: KeysConfig;
};

export type BranchRecord = {
  branchName: string;
  branchId: string;
  releaseChannel?: string | null;
  createdAt: string | null;
};

export type ChannelRecord = {
  releaseChannelId: string;
  releaseChannelName: string;
  branchName?: string | null;
  branchId?: string | null;
  createdAt: string | null;
};

export type ApiKeyRecord = {
  id: string;
  name: string;
  hint: string;
  createdAt: string;
  lastUsedAt?: string | null;
};

export type CreateApiKeyResponse = {
  apiKey: string;
};

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
      const contentType = response.headers.get('content-type');
      if (contentType && contentType.includes('application/problem+json')) {
        try {
          const problemPayload = (await response.json()) as APIProblemPayload;
          throw new ApiProblemError(problemPayload);
        } catch (parseError) {
          if (parseError instanceof ApiProblemError) throw parseError;
        }
      }
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

  public async createApp(payload: CreateAppPayload) {
    return this.request<{ appId: string }>(`/api/apps`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  }

  public async getApps() {
    return this.request<AppDescriptor[]>(`/api/apps`, {
      method: 'GET',
    });
  }

  public async getApp(appId: string) {
    return this.request<AppDetails>(`/api/apps/${encodeURIComponent(appId)}`, {
      method: 'GET',
    });
  }

  public async deleteApp() {
    return this.request<void>(`${this.appScope()}`, {
      method: 'DELETE',
    });
  }

  public async updateApp(payload: { name?: string }) {
    return this.request<void>(`${this.appScope()}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  }

  public async getApiKeys() {
    return this.request<ApiKeyRecord[]>(`${this.appScope()}/apiKeys`, {
      method: 'GET',
    });
  }

  public async createApiKey(name: string) {
    return this.request<CreateApiKeyResponse>(`${this.appScope()}/apiKeys`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
  }

  public async revokeApiKey(apiKeyId: string) {
    return this.request<void>(`${this.appScope()}/apiKeys/${encodeURIComponent(apiKeyId)}/revoke`, {
      method: 'DELETE',
    });
  }

  public async downloadAppCertificate(appId: string): Promise<string> {
    const url = `${this.baseUrl}/api/apps/${encodeURIComponent(appId)}/certificate`;
    const headers = new Headers();
    this.populateHeaders(headers);
    const response = await fetch(url, { method: 'GET', headers });
    const refreshToken = getRefreshToken();
    if (response.status === 401 && refreshToken) {
      await this.refreshTokens(refreshToken);
      return this.downloadAppCertificate(appId);
    }
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }
    return response.text();
  }

  public async getChannels() {
    return this.request<ChannelRecord[]>(`${this.appScope()}/channels`, {
      method: 'GET',
    });
  }

  public async createChannel(payload: { branchName?: string; channelName: string }) {
    return this.request<{ channelId: string }>(`${this.appScope()}/channels`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  }

  public async deleteChannel(channelName: string) {
    return this.request<void>(`${this.appScope()}/channels/${encodeURIComponent(channelName)}`, {
      method: 'DELETE',
    });
  }

  public async getBranches() {
    return this.request<BranchRecord[]>(`${this.appScope()}/branches`, {
      method: 'GET',
    });
  }

  public async createBranch(branchName: string) {
    return this.request<{ branchId: string }>(`${this.appScope()}/branches`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ branchName }),
    });
  }

  public async deleteBranch(branchName: string) {
    return this.request<void>(`${this.appScope()}/branches/${encodeURIComponent(branchName)}`, {
      method: 'DELETE',
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
      CONTROL_PLANE_ENABLED: boolean;
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
