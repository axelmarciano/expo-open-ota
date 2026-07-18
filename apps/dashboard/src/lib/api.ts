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

// Turns any thrown error into a toast-ready title/description pair: an
// ApiProblemError carries the server's actionable message, anything else
// falls back to the given title.
export const describeApiError = (
  error: unknown,
  fallbackTitle: string
): { title: string; description: string } => {
  if (error instanceof ApiProblemError) {
    return { title: error.title, description: error.detail };
  }
  return {
    title: fallbackTitle,
    description: error instanceof Error ? error.message : 'An unexpected error occurred.',
  };
};

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

// A dashboard user account. `id` is empty in stateless mode, where the only
// account comes from ADMIN_EMAIL and is not a database row. `lastConnectedAt`
// is absent until the account's first successful sign-in.
export type UserRecord = {
  id: string;
  email: string;
  isAdmin: boolean;
  createdAt?: string;
  lastConnectedAt?: string;
};

// The deployment's Enterprise Edition license status (/api/license,
// control-plane only). `valid` is the single source of truth for "enterprise
// features are on": `hasKey` can be true with `valid` false when the stored
// key is expired or malformed, in which case `error` says why. `expiresAt` is
// absent for a perpetual license.
export type LicenseStatus = {
  hasKey: boolean;
  valid: boolean;
  error?: string;
  licenseId?: string;
  issuedAt?: string;
  expiresAt?: string;
  activatedAt?: string;
};

// Pre-auth SSO state (/auth/sso/config), read by the login page to decide
// whether to render the SSO button. `enabled` is false for every possible
// reason at once (not configured, toggled off, no valid license, stateless).
export type SsoPublicConfig = {
  enabled: boolean;
  providerName?: string;
};

// Admin view of the SSO configuration (/api/sso). The client secret never
// leaves the server: `hasClientSecret` only says whether one is stored, and
// `redirectUri` is derived from BASE_URL for copy-pasting into the IdP.
export type SsoSettings = {
  issuer: string;
  clientId: string;
  hasClientSecret: boolean;
  providerName: string;
  scopes: string;
  enabled: boolean;
  allowedEmailDomains: string[];
  allowedGroups: string[];
  groupsClaim: string;
  redirectUri: string;
};

// An empty `clientSecret` on an update means "keep the stored secret".
export type SaveSsoSettingsPayload = {
  issuer: string;
  clientId: string;
  clientSecret?: string;
  providerName: string;
  scopes: string;
  enabled: boolean;
  allowedEmailDomains: string[];
  allowedGroups: string[];
  groupsClaim: string;
};

// Mirror of the server's SettingsEnv payload (/api/settings). Field names are
// the raw env-var spellings on purpose — the server is the source of truth.
export type ServerSettings = {
  BASE_URL: string;
  CONTROL_PLANE_ENABLED: boolean;
  CACHE_MODE: string;
  REDIS_HOST: string;
  REDIS_PORT: string;
  REDIS_SENTINEL_ADDRS: string;
  REDIS_SENTINEL_MASTER_NAME: string;
  STORAGE_MODE: string;
  S3_BUCKET_NAME: string;
  S3_CDN_PREFIX: string;
  GCS_BUCKET_NAME: string;
  LOCAL_BUCKET_BASE_PATH: string;
  AWS_REGION: string;
  AWS_BASE_ENDPOINT: string;
  AWS_S3_FORCE_PATH_STYLE: string;
  AWS_ACCESS_KEY_ID: string;
  CLOUDFRONT_DOMAIN: string;
  CLOUDFRONT_KEY_PAIR_ID: string;
  PRIVATE_CLOUDFRONT_KEY_B64: string;
  AWSSM_CLOUDFRONT_PRIVATE_KEY_SECRET_ID: string;
  PRIVATE_CLOUDFRONT_KEY_PATH: string;
  PROMETHEUS_ENABLED: string;
  CDN_TYPE: '' | 'cloudfront' | 'gcs-direct' | 's3-cdn-prefix';
  EXPO_ACCOUNT_USERNAME: string;
  SSO_ENABLED: boolean;
  APPS: { id: string; name?: string }[];
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
    // @ts-expect-error window.env is injected at runtime by /env.js
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

  public async login(email: string, password: string) {
    const form = new URLSearchParams();
    form.append('email', email);
    form.append('password', password);
    return this.request<{ token: string; refreshToken: string }>(`/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    });
  }

  public async getMe() {
    return this.request<UserRecord>(`/api/me`, {
      method: 'GET',
    });
  }

  public async changeMyPassword(payload: { currentPassword: string; newPassword: string }) {
    return this.request<void>(`/api/me/password`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  }

  public async getUsers() {
    return this.request<UserRecord[]>(`/api/users`, {
      method: 'GET',
    });
  }

  public async createUser(payload: { email: string; password: string; isAdmin: boolean }) {
    return this.request<UserRecord>(`/api/users`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  }

  public async updateUserAdmin(userId: string, isAdmin: boolean) {
    return this.request<void>(`/api/users/${encodeURIComponent(userId)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isAdmin }),
    });
  }

  public async deleteUser(userId: string) {
    return this.request<void>(`/api/users/${encodeURIComponent(userId)}`, {
      method: 'DELETE',
    });
  }

  public async getLicense() {
    return this.request<LicenseStatus>(`/api/license`, {
      method: 'GET',
    });
  }

  // Pre-auth: answers whether the SSO button should show on the login page.
  public async getSsoPublicConfig() {
    return this.request<SsoPublicConfig>(`/auth/sso/config`, {
      method: 'GET',
    });
  }

  // Entry point of the SSO flow: a plain navigation, not an XHR — the server
  // answers with a redirect to the identity provider.
  public ssoLoginUrl(): string {
    return `${this.baseUrl}/auth/sso/login`;
  }

  // The callback the IdP must allow. Derived the same way the server derives
  // it from BASE_URL; the server's value (SsoSettings.redirectUri) stays the
  // source of truth once a configuration exists.
  public ssoRedirectUri(): string {
    return `${this.baseUrl}/auth/sso/callback`;
  }

  // Admin SSO configuration. `null` means "not configured yet": the card
  // shows the empty form instead of an error.
  public async getSsoSettings(): Promise<SsoSettings | null> {
    try {
      return await this.request<SsoSettings>(`/api/sso`, { method: 'GET' });
    } catch (error) {
      if (error instanceof ApiProblemError && error.status === 404) {
        return null;
      }
      throw error;
    }
  }

  public async saveSsoSettings(payload: SaveSsoSettingsPayload) {
    return this.request<SsoSettings>(`/api/sso`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
  }

  public async deleteSsoSettings() {
    return this.request<void>(`/api/sso`, {
      method: 'DELETE',
    });
  }

  public async activateLicense(key: string) {
    return this.request<LicenseStatus>(`/api/license`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ key }),
    });
  }

  public async removeLicense() {
    return this.request<void>(`/api/license`, {
      method: 'DELETE',
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

  // Remaps a release channel onto a branch. The channel id drives the remap;
  // its name is also sent because the server invalidates the channel-mapping
  // cache by name.
  public async updateChannelBranchMapping(
    branchId: string,
    payload: {
      releaseChannelId: string;
      releaseChannelName: string;
    }
  ) {
    return this.request(
      `${this.appScope()}/branch/${encodeURIComponent(branchId)}/updateChannelBranchMapping`,
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
    return this.request<ServerSettings>(`/api/settings`, {
      method: 'GET',
    });
  }
}

export const api = new ApiClient();
