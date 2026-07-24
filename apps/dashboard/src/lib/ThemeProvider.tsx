import { ReactNode, useEffect, useState } from 'react';
import {
  applyTheme,
  getStoredPreference,
  resolveTheme,
  ResolvedTheme,
  storePreference,
  SYSTEM_DARK_QUERY,
  ThemeContext,
  ThemePreference,
} from '@/lib/theme';

export const ThemeProvider = ({ children }: { children: ReactNode }) => {
  const [preference, setPreferenceState] = useState<ThemePreference>(getStoredPreference);
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() =>
    resolveTheme(getStoredPreference())
  );

  useEffect(() => {
    const media = window.matchMedia(SYSTEM_DARK_QUERY);
    const syncTheme = () => {
      const next = resolveTheme(preference);
      setResolvedTheme(next);
      applyTheme(next);
    };
    syncTheme();
    if (preference !== 'system') return;
    media.addEventListener('change', syncTheme);
    return () => media.removeEventListener('change', syncTheme);
  }, [preference]);

  const setPreference = (next: ThemePreference) => {
    storePreference(next);
    setPreferenceState(next);
  };

  return (
    <ThemeContext.Provider value={{ preference, resolvedTheme, setPreference }}>
      {children}
    </ThemeContext.Provider>
  );
};
