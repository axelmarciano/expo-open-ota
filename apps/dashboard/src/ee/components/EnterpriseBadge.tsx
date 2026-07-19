// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { useQuery } from '@tanstack/react-query';
import { BadgeCheck } from 'lucide-react';
import { api } from '@/lib/api';
import { useSettings } from '@/lib/SettingsContext';

// Sidebar marker of an active Enterprise license. Renders nothing in
// stateless mode, while loading, or when no valid license is active; the
// community sidebar stays untouched. Shares the ['license'] query with the
// License page, so activating or removing a key updates it immediately.
export const EnterpriseBadge = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();

  const licenseQuery = useQuery({
    queryKey: ['license'],
    queryFn: () => api.getLicense(),
    enabled: CONTROL_PLANE_ENABLED,
  });

  const license = licenseQuery.data;
  if (!license?.valid) {
    return null;
  }

  return (
    <div className="mx-5 mt-2 flex items-center gap-2 rounded-lg border border-emerald-200/80 bg-gradient-to-r from-emerald-50/80 to-teal-50/60 px-2.5 py-1.5 shadow-sm">
      <BadgeCheck className="h-4 w-4 shrink-0 text-emerald-600" strokeWidth={2} />
      <div className="min-w-0 leading-tight">
        <div className="text-[11px] font-semibold text-emerald-900">Enterprise</div>
        <div className="truncate font-mono text-[10px] text-emerald-700/70" title={license.licenseId}>
          {license.licenseId?.split('-')[0]}
        </div>
      </div>
    </div>
  );
};
