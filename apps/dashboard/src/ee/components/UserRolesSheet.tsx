// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Sparkles, TriangleAlert } from 'lucide-react';
import { api, UserRecord, describeApiError } from '@/lib/api';
import { ApiError } from '@/components/APIError';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { EnterpriseExplainerDialog } from '@/ee/components/EnterpriseExplainerDialog';
import { DraftGrant, GrantsEditor } from '@/ee/components/GrantsEditor';

// The single "Roles" panel of a user row: the admin flag, and for members the
// per-app roles. The admin switch is a community feature and works without a
// license; the per-app editor is enterprise. Without a license it is replaced
// by a compact note (not the frosted overlay, which would read as the whole
// sheet being locked while the admin switch above stays usable).
// Everything applies on Save, in one place.
export const UserRolesSheet = ({
  user,
  onClose,
}: {
  user: UserRecord | null;
  onClose: () => void;
}) => {
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [isAdminDraft, setIsAdminDraft] = useState(false);
  const [draft, setDraft] = useState<DraftGrant[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [isExplainerOpen, setIsExplainerOpen] = useState(false);

  // Shares the ['license'] query with EnterpriseFeatureGate and the License
  // page, so activating a key swaps the note for the editor immediately.
  const licenseQuery = useQuery({
    queryKey: ['license'],
    queryFn: () => api.getLicense(),
  });
  const hasEnterpriseLicense = !!licenseQuery.data?.valid;

  const grantsQuery = useQuery({
    queryKey: ['userGrants', user?.id],
    queryFn: () => api.getUserGrants(user!.id),
    // Without a license the editor is not shown and grants are never written,
    // so there is nothing to fetch.
    enabled: !!user && hasEnterpriseLicense,
  });

  useEffect(() => {
    if (user) {
      setIsAdminDraft(user.isAdmin);
    }
    // Changing the target (or closing) must never carry the previous
    // member's draft over: the sheet stays mounted across opens, and a
    // failed fetch would otherwise show, and save, the last user's grants.
    setDraft([]);
  }, [user]);

  useEffect(() => {
    if (user && grantsQuery.data) {
      setDraft(
        grantsQuery.data.map(grant => ({
          appId: grant.appId,
          roleId: grant.roleId,
          extraPermissions: grant.extraPermissions,
        }))
      );
    }
  }, [user, grantsQuery.data]);

  const handleSave = async () => {
    if (!user) return;
    setIsSaving(true);
    try {
      if (isAdminDraft !== user.isAdmin) {
        await api.updateUserAdmin(user.id, isAdminDraft);
      }
      // Grants only matter for members; an admin's dormant grants are left
      // untouched so demoting them later restores their previous scope.
      // Without a license the server refuses grant writes, and community
      // rules apply anyway, so only the admin flag is saved.
      if (!isAdminDraft && hasEnterpriseLicense) {
        await api.setUserGrants(user.id, draft);
      }
      toast({
        title: 'Roles saved',
        description: isAdminDraft
          ? `"${user.email}" is an admin with full access.`
          : !hasEnterpriseLicense
            ? `"${user.email}" is a member.`
            : draft.length === 0
              ? `"${user.email}" has no app access.`
              : `"${user.email}" has access to ${draft.length} app${draft.length > 1 ? 's' : ''}.`,
      });
      queryClient.invalidateQueries({ queryKey: ['users'] });
      queryClient.invalidateQueries({ queryKey: ['userGrants', user.id] });
      queryClient.invalidateQueries({ queryKey: ['userGrantsSummary'] });
      onClose();
    } catch (error) {
      toast({ ...describeApiError(error, 'Could not save roles'), variant: 'destructive' });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Sheet open={!!user} onOpenChange={open => !open && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Roles</SheetTitle>
          <SheetDescription>What “{user?.email}” can see and do.</SheetDescription>
        </SheetHeader>

        <div className="mt-6 space-y-5">
          <label className="flex cursor-pointer items-center justify-between gap-3 rounded-xl border p-4 text-sm">
            <span>
              <span className="font-medium">Administrator</span>
              <span className="block text-xs text-muted-foreground">
                Full access to every app; manages users, roles and server settings.
              </span>
            </span>
            <Switch
              checked={isAdminDraft}
              disabled={isSaving}
              onCheckedChange={setIsAdminDraft}
              aria-label="Administrator"
            />
          </label>

          {isAdminDraft ? (
            <p className="rounded-lg border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
              Admins bypass roles entirely, so there is nothing else to configure.
            </p>
          ) : !hasEnterpriseLicense ? (
            <div className="space-y-3 rounded-lg border bg-muted/30 px-3 py-2.5 text-xs text-muted-foreground">
              <p>
                Members can view every app, read-only. Per-app roles and permissions are an
                Enterprise feature.
              </p>
              <Button variant="outline" size="sm" onClick={() => setIsExplainerOpen(true)}>
                <Sparkles className="h-3.5 w-3.5" />
                Discover Enterprise
              </Button>
            </div>
          ) : grantsQuery.isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-4 w-1/2" />
            </div>
          ) : grantsQuery.isError ? (
            // A failed fetch must never render as "no grants": with an
            // empty draft, Save would wipe the member's real grants.
            <div className="space-y-3">
              <ApiError error={grantsQuery.error} />
              <Button variant="outline" size="sm" onClick={() => grantsQuery.refetch()}>
                Try again
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {draft.length === 0 && (
                <div className="flex items-start gap-2.5 rounded-xl border border-amber-300/60 bg-amber-50 px-4 py-3 text-xs text-amber-800">
                  <TriangleAlert className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                  <span>
                    This member has no app access: they will see an empty dashboard. Grant them an
                    app below, with a role.
                  </span>
                </div>
              )}
              <GrantsEditor draft={draft} onChange={setDraft} disabled={isSaving} />
            </div>
          )}

          <div className="flex justify-end gap-2 border-t pt-4">
            <Button variant="outline" onClick={onClose} disabled={isSaving}>
              Cancel
            </Button>
            <Button
              onClick={handleSave}
              // isSuccess, not !isLoading: saving grants over a failed fetch
              // would replace the member's set with the empty draft. Without
              // a license grants are never fetched nor written, so only the
              // save in flight blocks the button.
              disabled={
                isSaving || (!isAdminDraft && hasEnterpriseLicense && !grantsQuery.isSuccess)
              }>
              {isSaving ? 'Saving…' : 'Save'}
            </Button>
          </div>
        </div>

        <EnterpriseExplainerDialog open={isExplainerOpen} onOpenChange={setIsExplainerOpen} />
      </SheetContent>
    </Sheet>
  );
};
