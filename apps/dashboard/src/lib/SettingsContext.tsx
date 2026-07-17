import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api, ApiProblemError } from '@/lib/api';

export interface Settings {
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
}

const SettingsContext = createContext<Settings | null>(null);

interface SettingsProviderProps {
  children: ReactNode;
}

export function SettingsProvider({ children }: SettingsProviderProps) {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchSettings() {
        try {
            const data = await api.getSettings();
            setSettings(data);
        } 
        catch (error) {
            let errorMessage = 'An unexpected server error occurred.';
            if (error instanceof ApiProblemError) {
                errorMessage = error.detail;
            }
            setError(errorMessage);
        } finally {
            setLoading(false);
        }
    }
    fetchSettings();
  }, []);

  if (loading) {
    return <div className="loading-screen">Loading dashboard configurations...</div>;
  }

  if (error) {
    return <div className="error-screen">Error: {error}</div>;
  }

  return (
    <SettingsContext.Provider value={settings}>
      {children}
    </SettingsContext.Provider>
  );
}

export function useSettings(): Settings {
  const context = useContext(SettingsContext);
  if (!context) {
    throw new Error('useSettings must be used within a SettingsProvider');
  }
  return context;
}