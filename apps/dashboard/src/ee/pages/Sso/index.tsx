// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { useSettings } from '@/lib/SettingsContext';
import { useCurrentUser } from '@/lib/CurrentUserContext';
import { PageHeader } from '@/components/PageHeader';
import { EnterpriseFeatureGate } from '@/ee/components/EnterpriseFeatureGate';
import { SsoConfigCard } from '@/ee/components/SsoConfigCard';

// The SSO page of the Access & Security sidebar group. Without a valid
// license the configuration card stays visible behind the frosted enterprise
// gate, following the same convention as the other enterprise blocks.
export const Sso = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  const { isAdmin } = useCurrentUser();

  if (!CONTROL_PLANE_ENABLED) {
    return (
      <div className="w-full">
        <PageHeader
          title="Single sign-on"
          description="Let your team sign in with your identity provider."
        />
        <div className="rounded-xl border border-dashed bg-muted/30 p-8 text-center text-sm text-muted-foreground">
          Single sign-on stores its configuration and its user accounts in the database, so it
          requires control-plane (DB) mode. Stateless deployments sign in with ADMIN_EMAIL and
          ADMIN_PASSWORD.
        </div>
      </div>
    );
  }

  if (!isAdmin) {
    return (
      <div className="w-full">
        <PageHeader
          title="Single sign-on"
          description="Let your team sign in with your identity provider."
        />
        <div className="rounded-xl border border-dashed bg-muted/30 p-8 text-center text-sm text-muted-foreground">
          Only admins can configure single sign-on. Ask an admin if you need access.
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <PageHeader
        title="Single sign-on"
        description="Let your team sign in through your identity provider. Accounts are created automatically as members on their first sign-in, and admins keep their password as a break-glass access."
      />
      <EnterpriseFeatureGate>
        <SsoConfigCard />
      </EnterpriseFeatureGate>
    </div>
  );
};
