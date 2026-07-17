// Copyright (c) 2026 Mercure Technologies. All rights reserved.
// This file is governed by the Expo Open OTA Enterprise Edition license
// (see ee/LICENSE at the repository root); it is NOT covered by the MIT
// license of this repository.

import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiKeyRecord, ApiKeyScopeRecord, ChannelRecord, describeApiError } from '@/lib/api';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { EnterpriseFeatureGate } from '@/ee/components/EnterpriseFeatureGate';

// Side panel to edit the enterprise access restrictions of one API token:
// which release channels it may publish to and which source addresses may use
// it. Without a valid license the form is masked by EnterpriseFeatureGate.
export const ApiKeyRestrictionsSheet = ({
  apiKey,
  onClose,
}: {
  apiKey: ApiKeyRecord | null;
  onClose: () => void;
}) => {
  const { selectedAppId } = useSelectedApp();

  const scopesQuery = useQuery({
    queryKey: ['apiKeyScopes', selectedAppId],
    queryFn: () => api.getApiKeyScopes(),
    enabled: !!selectedAppId,
  });

  const channelsQuery = useQuery({
    queryKey: ['channels', selectedAppId],
    queryFn: () => api.getChannels(),
    enabled: !!selectedAppId,
  });

  const isLoading = scopesQuery.isLoading || channelsQuery.isLoading;

  return (
    <Sheet open={!!apiKey} onOpenChange={open => !open && onClose()}>
      <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-md">
        <SheetHeader>
          <SheetTitle>Access restrictions</SheetTitle>
          <SheetDescription>
            Scope “{apiKey?.name}” to specific release channels, or whitelist the IP addresses
            allowed to use it. Without restrictions, the token can publish to every channel from
            any address.
          </SheetDescription>
        </SheetHeader>
        <div className="mt-6">
          <EnterpriseFeatureGate>
            {isLoading ? (
              <div className="space-y-3">
                <Skeleton className="h-4 w-1/2" />
                <Skeleton className="h-24 w-full" />
                <Skeleton className="h-4 w-1/2" />
                <Skeleton className="h-24 w-full" />
              </div>
            ) : (
              apiKey && (
                <RestrictionsForm
                  // Remount when switching tokens so the form state resets to
                  // the stored restrictions of the newly opened token.
                  key={apiKey.id}
                  apiKey={apiKey}
                  channels={channelsQuery.data ?? []}
                  initialScopes={scopesQuery.data?.find(scope => scope.apiKeyId === apiKey.id)}
                  onSaved={onClose}
                />
              )
            )}
          </EnterpriseFeatureGate>
        </div>
      </SheetContent>
    </Sheet>
  );
};

const RestrictionsForm = ({
  apiKey,
  channels,
  initialScopes,
  onSaved,
}: {
  apiKey: ApiKeyRecord;
  channels: ChannelRecord[];
  initialScopes?: ApiKeyScopeRecord;
  onSaved: () => void;
}) => {
  const { selectedAppId } = useSelectedApp();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [selectedChannelIds, setSelectedChannelIds] = useState<string[]>(
    initialScopes?.channelIds ?? []
  );
  const [allowedIpsText, setAllowedIpsText] = useState(
    (initialScopes?.allowedIps ?? []).join('\n')
  );
  const [isSaving, setIsSaving] = useState(false);

  const toggleChannel = (channelId: string) => {
    setSelectedChannelIds(current =>
      current.includes(channelId)
        ? current.filter(id => id !== channelId)
        : [...current, channelId]
    );
  };

  const handleSave = async () => {
    setIsSaving(true);
    try {
      const allowedIps = allowedIpsText
        .split('\n')
        .map(line => line.trim())
        .filter(Boolean);
      await api.setApiKeyScopes(apiKey.id, { channelIds: selectedChannelIds, allowedIps });
      queryClient.invalidateQueries({ queryKey: ['apiKeyScopes', selectedAppId] });
      toast({
        title: 'Restrictions saved',
        description: `“${apiKey.name}” now uses the updated restrictions.`,
      });
      onSaved();
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not save restrictions');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <p className="text-sm font-medium">Release channels</p>
        <p className="text-xs text-muted-foreground">
          The token can only publish to the selected channels. Leave everything unchecked to allow
          every channel.
        </p>
        {channels.length === 0 ? (
          <div className="rounded-lg border border-dashed bg-muted/30 p-4 text-center text-xs text-muted-foreground">
            No channels yet. Create one on the Channels page to scope this token.
          </div>
        ) : (
          <div className="space-y-1.5">
            {channels.map(channel => (
              <label
                key={channel.releaseChannelId}
                className="flex cursor-pointer items-center gap-2.5 rounded-lg border px-3 py-2 transition-colors hover:bg-muted/40">
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-input accent-emerald-600"
                  checked={selectedChannelIds.includes(channel.releaseChannelId)}
                  onChange={() => toggleChannel(channel.releaseChannelId)}
                />
                <span className="text-sm font-medium">{channel.releaseChannelName}</span>
                {channel.branchName && (
                  <span className="ml-auto text-xs text-muted-foreground">
                    → {channel.branchName}
                  </span>
                )}
              </label>
            ))}
          </div>
        )}
      </div>

      <div className="space-y-2">
        <p className="text-sm font-medium">IP allowlist</p>
        <p className="text-xs text-muted-foreground">
          One address or CIDR range per line, for example 203.0.113.7 or 203.0.113.0/24. Leave
          empty to allow any source address.
        </p>
        <textarea
          value={allowedIpsText}
          onChange={event => setAllowedIpsText(event.target.value)}
          placeholder={'203.0.113.0/24\n2001:db8::/32'}
          rows={5}
          spellCheck={false}
          className="w-full resize-y rounded-md border border-input bg-transparent px-3 py-2 font-mono text-xs shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
      </div>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={isSaving}>
          {isSaving ? 'Saving…' : 'Save restrictions'}
        </Button>
      </div>
    </div>
  );
};
