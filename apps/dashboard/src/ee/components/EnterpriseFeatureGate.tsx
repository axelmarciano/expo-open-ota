// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { ReactNode, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { Lock, Mail, Sparkles } from 'lucide-react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

const CONTACT_EMAIL = 'contact@mercuretechnologies.com';

// Wraps an enterprise-only block. With a valid license the children render
// untouched. Without one they stay visible (never hidden) but inert behind a
// frosted overlay with an upsell card; its button opens a dialog explaining
// how to get a license. Shares the ['license'] query with the License page
// and EnterpriseBadge, so activating a key unlocks the block immediately.
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
      <div aria-hidden className="pointer-events-none select-none opacity-60">
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
              Unlock it with an Expo Open OTA Enterprise license.
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

      <Dialog open={isExplainerOpen} onOpenChange={setIsExplainerOpen}>
        <DialogContent>
          <DialogHeader>
            <div className="mb-1 flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 shadow-sm">
              <Lock className="h-5 w-5 text-white" strokeWidth={2.2} />
            </div>
            <DialogTitle>Unlock Enterprise features</DialogTitle>
            <DialogDescription>
              This feature is part of the Enterprise edition of Expo Open OTA. Your deployment
              currently runs the community edition, so it stays visible here but locked.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3 text-sm text-muted-foreground">
            <p>
              Want to know more or get a license? Write to us at{' '}
              <a href={`mailto:${CONTACT_EMAIL}`} className="font-medium text-link hover:underline">
                {CONTACT_EMAIL}
              </a>{' '}
              and we will get back to you quickly.
            </p>
            <p>
              Already have a key? Activate it on the{' '}
              <Link
                to="/license"
                onClick={() => setIsExplainerOpen(false)}
                className="font-medium text-link hover:underline">
                License page
              </Link>
              .
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsExplainerOpen(false)}>
              Close
            </Button>
            <Button
              asChild
              className="bg-gradient-to-r from-emerald-600 to-teal-600 text-white hover:from-emerald-700 hover:to-teal-700">
              <a href={`mailto:${CONTACT_EMAIL}?subject=Expo%20Open%20OTA%20Enterprise`}>
                <Mail className="h-4 w-4" />
                Contact us
              </a>
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
};
