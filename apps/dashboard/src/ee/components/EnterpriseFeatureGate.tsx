// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { ReactNode, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Lock, Sparkles } from 'lucide-react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { EnterpriseExplainerDialog } from '@/ee/components/EnterpriseExplainerDialog';

// Wraps an enterprise-only block. With a valid license the children render
// untouched. Without one they stay visible (never hidden) but inert behind a
// frosted overlay with an upsell card; its button opens the enterprise
// explainer dialog. Shares the ['license'] query with the License page and
// EnterpriseBadge, so activating a key unlocks the block immediately.
export const EnterpriseFeatureGate = ({ children }: { children: ReactNode }) => {
  const [isExplainerOpen, setIsExplainerOpen] = useState(false);

  const licenseQuery = useQuery({
    queryKey: ['license'],
    queryFn: () => api.getLicense(),
  });

  if (licenseQuery.data?.valid) {
    return <>{children}</>;
  }

  return (
    <div className="relative">
      {/* react-dom 18 drops inert={true}; the attribute is only set when given a
          string, hence the cast (our @types/react is v19, which types it as boolean). */}
      <div
        aria-hidden
        inert={'' as unknown as boolean}
        className="pointer-events-none select-none opacity-60">
        {children}
      </div>
      <div className="absolute inset-0 z-10 flex items-center justify-center rounded-xl bg-white/55 backdrop-blur-[3px]">
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-emerald-100 bg-white px-8 py-6 text-center shadow-elevated">
          <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 shadow-sm">
            <Lock className="h-5 w-5 text-white" strokeWidth={2.2} />
          </div>
          <div>
            <p className="text-sm font-semibold">Enterprise feature</p>
            <p className="mt-1 max-w-[230px] text-xs leading-relaxed text-muted-foreground">
              Unlock it with an Enterprise license.
            </p>
          </div>
          <Button
            size="sm"
            onClick={() => setIsExplainerOpen(true)}
            className="bg-gradient-to-r from-emerald-600 to-teal-600 text-white shadow-sm hover:from-emerald-700 hover:to-teal-700">
            <Sparkles className="h-3.5 w-3.5" />
            Discover Enterprise
          </Button>
        </div>
      </div>

      <EnterpriseExplainerDialog open={isExplainerOpen} onOpenChange={setIsExplainerOpen} />
    </div>
  );
};
