// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { BadgeCheck, FileUp, ShieldAlert } from 'lucide-react';
import { api, describeApiError } from '@/lib/api';
import { useSettings } from '@/lib/SettingsContext';
import { useCurrentUser } from '@/lib/CurrentUserContext';
import { useToast } from '@/hooks/use-toast';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { PageHeader } from '@/components/PageHeader';
import { Skeleton } from '@/components/ui/skeleton';
import { TimestampCell } from '@/components/ui/timestamp-cell';

const StatusRow = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="flex items-center justify-between gap-4 py-2 text-sm">
    <span className="text-muted-foreground">{label}</span>
    <span>{children}</span>
  </div>
);

export const License = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  const { isAdmin } = useCurrentUser();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [keyInput, setKeyInput] = useState('');
  const [isActivating, setIsActivating] = useState(false);
  const [isRemoveDialogOpen, setIsRemoveDialogOpen] = useState(false);
  const [isRemoving, setIsRemoving] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const licenseQuery = useQuery({
    queryKey: ['license'],
    queryFn: () => api.getLicense(),
    enabled: CONTROL_PLANE_ENABLED,
  });

  const notifyError = (error: unknown, fallbackTitle: string) =>
    toast({ ...describeApiError(error, fallbackTitle), variant: 'destructive' });

  const handleActivate = async () => {
    const key = keyInput.trim();
    if (!key) return;
    setIsActivating(true);
    try {
      const status = await api.activateLicense(key);
      queryClient.invalidateQueries({ queryKey: ['license'] });
      setKeyInput('');
      toast({
        title: 'License activated',
        description: status.expiresAt
          ? `Enterprise edition is enabled until ${new Date(status.expiresAt).toLocaleDateString()}.`
          : 'Enterprise edition is enabled (perpetual license).',
      });
    } catch (error) {
      notifyError(error, 'License activation failed');
    } finally {
      setIsActivating(false);
    }
  };

  const handleImportFile = async (file: File | undefined) => {
    if (!file) return;
    setKeyInput((await file.text()).trim());
    // Reset so picking the same file again still fires onChange.
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const handleRemove = async () => {
    setIsRemoving(true);
    try {
      await api.removeLicense();
      queryClient.invalidateQueries({ queryKey: ['license'] });
      setIsRemoveDialogOpen(false);
      toast({
        title: 'License removed',
        description: 'This deployment is back to community edition.',
      });
    } catch (error) {
      notifyError(error, 'License removal failed');
    } finally {
      setIsRemoving(false);
    }
  };

  if (!CONTROL_PLANE_ENABLED) {
    return (
      <div className="w-full">
        <PageHeader
          title="License"
          description="Enterprise Edition license for this deployment."
        />
        <div className="rounded-xl border border-dashed bg-muted/30 p-8 text-center text-sm text-muted-foreground">
          Enterprise licenses are stored in the database and require control-plane (DB) mode.
          Stateless deployments run the community edition.
        </div>
      </div>
    );
  }

  const license = licenseQuery.data;

  const activationForm = (
    <div className="space-y-3">
      <textarea
        value={keyInput}
        onChange={(event) => setKeyInput(event.target.value)}
        placeholder="key/…"
        rows={4}
        spellCheck={false}
        className="w-full resize-y rounded-md border border-input bg-transparent px-3 py-2 font-mono text-xs shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      />
      <div className="flex items-center gap-2">
        <Button onClick={handleActivate} disabled={!keyInput.trim() || isActivating}>
          {isActivating ? 'Verifying…' : 'Verify & activate'}
        </Button>
        <Button variant="outline" onClick={() => fileInputRef.current?.click()}>
          <FileUp className="h-4 w-4" />
          Import from file
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          accept=".txt,.key,.lic,text/plain"
          className="hidden"
          onChange={(event) => handleImportFile(event.target.files?.[0])}
        />
      </div>
    </div>
  );

  return (
    <div className="w-full">
      <PageHeader
        title="License"
        description="Enterprise Edition license for this deployment. Keys are verified offline against the Expo Open OTA signing key — no phone home — and stored in the database."
      />

      {licenseQuery.isLoading && (
        <Card>
          <CardHeader>
            <Skeleton className="h-5 w-40" />
          </CardHeader>
          <CardContent className="space-y-2">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-2/3" />
          </CardContent>
        </Card>
      )}

      {!licenseQuery.isLoading && (
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                {license?.valid ? (
                  <>
                    <BadgeCheck className="h-5 w-5 text-emerald-600" />
                    Enterprise edition
                    <Badge>Active</Badge>
                  </>
                ) : (
                  <>
                    Community edition
                    <Badge variant="secondary">No active license</Badge>
                  </>
                )}
              </CardTitle>
              {license?.hasKey && !license.valid && (
                <CardDescription className="flex items-center gap-1.5 text-destructive">
                  <ShieldAlert className="h-4 w-4 shrink-0" />
                  {license.error ?? 'The stored license key is not usable.'}
                </CardDescription>
              )}
            </CardHeader>
            {license?.hasKey && license.licenseId && (
              <CardContent>
                <div className="divide-y">
                  <StatusRow label="License ID">
                    <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{license.licenseId}</code>
                  </StatusRow>
                  <StatusRow label="Issued">
                    <TimestampCell dateString={license.issuedAt ?? null} />
                  </StatusRow>
                  <StatusRow label="Expires">
                    {license.expiresAt ? (
                      <TimestampCell dateString={license.expiresAt} />
                    ) : (
                      <span className="text-muted-foreground">Never (perpetual)</span>
                    )}
                  </StatusRow>
                  <StatusRow label="Activated">
                    <TimestampCell dateString={license.activatedAt ?? null} />
                  </StatusRow>
                </div>
                {isAdmin && (
                  <div className="mt-4 flex justify-end">
                    <Button variant="outline" onClick={() => setIsRemoveDialogOpen(true)}>
                      Remove license
                    </Button>
                  </div>
                )}
              </CardContent>
            )}
          </Card>

          {isAdmin ? (
            <Card>
              <CardHeader>
                <CardTitle>{license?.valid ? 'Replace license key' : 'Activate a license key'}</CardTitle>
                <CardDescription>
                  Paste the license key you received with your Enterprise subscription, or import
                  the file it was delivered in. The key is verified before anything is stored.
                </CardDescription>
              </CardHeader>
              <CardContent>{activationForm}</CardContent>
            </Card>
          ) : (
            <div className="rounded-xl border border-dashed bg-muted/30 p-6 text-center text-sm text-muted-foreground">
              Only admins can activate or remove a license key.
            </div>
          )}
        </div>
      )}

      <DeleteDialog
        isOpen={isRemoveDialogOpen}
        onClose={() => setIsRemoveDialogOpen(false)}
        onConfirm={handleRemove}
        isDeleting={isRemoving}
        title="Remove license"
        resourceName={license?.licenseId}
        descriptionText="Enterprise features will be disabled and this deployment drops back to community edition. You can re-activate the same key later."
        confirmButtonText="Remove license"
        isDeletingButtonText="Removing…"
      />
    </div>
  );
};
