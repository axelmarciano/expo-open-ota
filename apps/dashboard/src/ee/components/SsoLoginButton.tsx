// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { useQuery } from '@tanstack/react-query';
import { KeyRound } from 'lucide-react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';

// The "Continue with <provider>" entry on the login page. Renders nothing
// while loading or whenever SSO is unavailable (not configured, disabled, no
// license, stateless mode), so the community login form stays untouched. The
// sign-in itself is a plain navigation: the server redirects to the IdP.
export const SsoLoginButton = () => {
  const configQuery = useQuery({
    queryKey: ['ssoPublicConfig'],
    queryFn: () => api.getSsoPublicConfig(),
    staleTime: 60_000,
    retry: false,
  });

  if (!configQuery.data?.enabled) {
    return null;
  }
  const providerName = configQuery.data.providerName || 'SSO';

  return (
    <div className="mt-5">
      <div className="relative">
        <div className="absolute inset-0 flex items-center">
          <span className="w-full border-t" />
        </div>
        <div className="relative flex justify-center text-xs">
          <span className="bg-background px-2 text-muted-foreground">or</span>
        </div>
      </div>
      <Button asChild variant="outline" className="mt-5 w-full">
        <a href={api.ssoLoginUrl()}>
          <KeyRound className="h-4 w-4" />
          Continue with {providerName}
        </a>
      </Button>
    </div>
  );
};
