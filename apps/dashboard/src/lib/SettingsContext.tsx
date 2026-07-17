import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { api, ApiProblemError, ServerSettings } from '@/lib/api';

export type Settings = ServerSettings;

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
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
        Loading dashboard…
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center px-4">
        <div className="max-w-md rounded-xl border border-destructive/30 bg-destructive/5 p-6 text-center">
          <p className="text-sm font-medium text-destructive">Could not reach the server</p>
          <p className="mt-1 text-sm text-muted-foreground">{error}</p>
        </div>
      </div>
    );
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