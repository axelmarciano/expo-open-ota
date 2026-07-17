import { createContext, useContext, ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api, UserRecord } from '@/lib/api';

type CurrentUserContextValue = {
  user: UserRecord | null;
  isAdmin: boolean;
};

const CurrentUserContext = createContext<CurrentUserContextValue | null>(null);

// CurrentUserProvider resolves the account behind the session (/api/me) so the
// UI can hide admin-only actions — creating apps, remapping channels, managing
// users. This is display gating only: the server re-checks the admin flag on
// every admin route.
export function CurrentUserProvider({ children }: { children: ReactNode }) {
  const { data, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: () => api.getMe(),
  });

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-muted-foreground">
        Loading dashboard…
      </div>
    );
  }

  return (
    <CurrentUserContext.Provider value={{ user: data ?? null, isAdmin: data?.isAdmin ?? false }}>
      {children}
    </CurrentUserContext.Provider>
  );
}

export function useCurrentUser(): CurrentUserContextValue {
  const context = useContext(CurrentUserContext);
  if (!context) {
    throw new Error('useCurrentUser must be used within a CurrentUserProvider');
  }
  return context;
}
