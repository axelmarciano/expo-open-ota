import { useMutation, useQuery } from '@tanstack/react-query';
import { api, ChannelRecord, ApiProblemError } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { SelectBranch } from '@/pages/Channels/components/SelectBranch';
import { useCallback, useMemo, useState } from 'react';
import { useToast } from '@/hooks/use-toast.ts';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { PageHeader } from '@/components/PageHeader';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { AdminOnlyNote } from '@/components/ui/admin-only-note';
import { TimestampCell } from '@/components/ui/timestamp-cell';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { Trash2, Plus, Split, Lock } from 'lucide-react';
import { useSettings } from '@/lib/SettingsContext';
import { useCurrentUser } from '@/lib/CurrentUserContext';
import { RolloutBar } from '@/components/rollout/RolloutBar';
import { StartRolloutDialog } from '@/pages/Channels/components/StartRolloutDialog';
import { ManageRolloutDialog } from '@/pages/Channels/components/ManageRolloutDialog';

interface TableColumnConfig {
  header: string;
  accessorKey?: keyof ChannelRecord;
  id?: string;
  cell: (props: { row: { original: ChannelRecord } }) => React.ReactNode;
}

export const Channels = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  const { isAdmin } = useCurrentUser();
  const { selectedAppId } = useSelectedApp();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['channels', selectedAppId],
    enabled: !!selectedAppId,
    queryFn: () => api.getChannels(),
  });
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);
  const [newChannelName, setNewChannelName] = useState('');
  const [newChannelBranch, setNewChannelBranch] = useState<{ id: string; name: string } | null>(
    null
  );
  const [isCreating, setIsCreating] = useState(false);
  const [channelToDelete, setChannelToDelete] = useState<ChannelRecord | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [rolloutAction, setRolloutAction] = useState<{
    type: 'start' | 'manage';
    channel: ChannelRecord;
  } | null>(null);

  const handleRolloutDone = useCallback(async () => {
    await refetch();
  }, [refetch]);

  // The dialogs must render the live record, not the snapshot captured when the
  // action was opened: after "Save percentage" the refetched channels list is the
  // source of truth, and a stale snapshot would show the pre-save value.
  const rolloutActionChannel = useMemo(() => {
    if (!rolloutAction) return null;
    return (
      data?.find(channel => channel.releaseChannelId === rolloutAction.channel.releaseChannelId) ??
      rolloutAction.channel
    );
  }, [rolloutAction, data]);

  const updateBranchMutation = useMutation({
    mutationKey: ['update-branch'],
    mutationFn: async ({
      branchId,
      releaseChannelId,
      releaseChannelName,
    }: {
      branchId: string;
      releaseChannelId: string;
      releaseChannelName: string;
    }) => {
      return api.updateChannelBranchMapping(branchId, {
        releaseChannelId,
        releaseChannelName,
      });
    },
  });

  const onBranchChange = useCallback(
    (channel: ChannelRecord) => async (branchId?: string | null) => {
      if (!branchId) return;
      setLoading(true);
      try {
        await updateBranchMutation.mutateAsync({
          branchId,
          releaseChannelId: channel.releaseChannelId,
          releaseChannelName: channel.releaseChannelName,
        });
        await refetch();
        toast({
          title: 'Channel updated',
          description: `Branch mapping synchronized successfully.`,
          duration: 2000,
        });
      } catch (error) {
        let errorTitle = 'Error updating channel mapping';
        let errorMessage = 'An unexpected error occurred.';
        if (error instanceof ApiProblemError) {
          errorTitle = error.title;
          errorMessage = error.detail;
        } else if (error instanceof Error) {
          errorMessage = error.message;
        }
        toast({
          title: errorTitle,
          description: errorMessage,
          variant: 'destructive',
        });
      } finally {
        setLoading(false);
      }
    },
    [updateBranchMutation, toast, refetch]
  );

  const handleCreateChannel = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newChannelName.trim()) return;
    setIsCreating(true);
    try {
      await api.createChannel({
        channelName: newChannelName.trim(),
        ...(newChannelBranch && { branchName: newChannelBranch.name }),
      });
      setNewChannelName('');
      setNewChannelBranch(null);
      await refetch();
      toast({
        title: 'Channel created',
        description: newChannelBranch
          ? `"${newChannelName.trim()}" now serves the "${newChannelBranch.name}" branch.`
          : `"${newChannelName.trim()}" created. Map it to a branch to start serving updates.`,
      });
    } catch (error) {
      let errorTitle = 'Error creating channel';
      let errorMessage = 'An unexpected error occurred.';
      if (error instanceof ApiProblemError) {
        errorTitle = error.title;
        errorMessage = error.detail;
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }
      toast({
        title: errorTitle,
        description: errorMessage,
        variant: 'destructive',
      });
    } finally {
      setIsCreating(false);
    }
  };

  const handleExecuteDeletion = async () => {
    if (!channelToDelete) return;
    setIsDeleting(true);
    try {
      await api.deleteChannel(channelToDelete.releaseChannelName);
      await refetch();
      toast({
        title: 'Channel deleted',
        description: `"${channelToDelete.releaseChannelName}" was removed.`,
      });
      setChannelToDelete(null);
    } catch (error) {
      let errorTitle = 'Deletion Failed';
      let errorMessage = 'Could not clean up the requested release channel.';
      if (error instanceof ApiProblemError) {
        errorTitle = error.title;
        errorMessage = error.detail;
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }
      toast({
        title: errorTitle,
        description: errorMessage,
        variant: 'destructive',
      });
    } finally {
      setIsDeleting(false);
    }
  };

  const tableColumns = useMemo<TableColumnConfig[]>(() => {
    return [
      {
        header: 'Channel',
        accessorKey: 'releaseChannelName',
        cell: ({ row }) => <span className="font-medium">{row.original.releaseChannelName}</span>,
      },
      {
        header: 'Branch',
        accessorKey: 'branchId',
        // Remapping a live channel is an admin action (the server enforces
        // it); everyone else sees the mapping read-only. While a rollout is
        // active the mapping is locked for everyone: it can only change by
        // promoting or reverting the rollout.
        cell: ({ row }) => {
          if (row.original.rollout) {
            return (
              <TooltipProvider delayDuration={200}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex items-center gap-1.5 text-sm text-muted-foreground">
                      <Lock className="h-3.5 w-3.5" />
                      {row.original.branchName || 'No branch'}
                    </span>
                  </TooltipTrigger>
                  <TooltipContent className="max-w-xs">
                    The branch mapping is locked while a rollout is in progress. Promote or revert
                    the rollout to change it.
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            );
          }
          return isAdmin ? (
            <SelectBranch
              currentBranch={row.original.branchId || ''}
              loading={isLoading || loading}
              onChange={onBranchChange(row.original)}
            />
          ) : (
            <span className="text-muted-foreground">{row.original.branchName || 'No branch'}</span>
          );
        },
      },
      ...(CONTROL_PLANE_ENABLED
        ? [
            {
              header: 'Rollout',
              id: 'rollout',
              cell: ({ row }) => {
                const channel = row.original;
                if (channel.rollout) {
                  return (
                    <div className="flex items-center gap-2.5">
                      <RolloutBar value={channel.rollout.percentage} />
                      <span className="text-xs text-muted-foreground">
                        to {channel.rollout.rolloutBranchName}
                      </span>
                      {isAdmin && (
                        <button
                          type="button"
                          onClick={() => setRolloutAction({ type: 'manage', channel })}
                          className="text-sm font-medium text-link hover:underline">
                          Manage
                        </button>
                      )}
                    </div>
                  );
                }
                // Nothing to roll out until the channel serves a branch.
                if (!channel.branchId) {
                  return <span className="text-muted-foreground/60">None</span>;
                }
                if (isAdmin) {
                  return (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setRolloutAction({ type: 'start', channel })}
                      className="text-muted-foreground hover:text-foreground">
                      <Split className="h-4 w-4" />
                      Start rollout
                    </Button>
                  );
                }
                return <span className="text-muted-foreground/60">None</span>;
              },
            } satisfies TableColumnConfig,
            {
              header: 'Created',
              accessorKey: 'createdAt',
              cell: ({ row }) => <TimestampCell dateString={row.original.createdAt} />,
            } satisfies TableColumnConfig,
          ]
        : []),
      // Deleting a channel is an admin action (the server enforces it).
      ...(CONTROL_PLANE_ENABLED && isAdmin
        ? [
            {
              header: '',
              id: 'actions',
              cell: ({ row }) => (
                <div className="text-right pr-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setChannelToDelete(row.original)}
                    className="h-8 w-8 p-0 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                    title="Delete Release Channel">
                    <Trash2 />
                  </Button>
                </div>
              ),
            } satisfies TableColumnConfig,
          ]
        : []),
    ];
  }, [CONTROL_PLANE_ENABLED, isAdmin, isLoading, loading, onBranchChange]);

  return (
    <div className="w-full">
      <PageHeader
        title="Channels"
        description={
          <>
            <p>
              A <span className="font-medium text-foreground">branch</span> is a line of updates you
              publish to, much like a git branch. A{' '}
              <span className="font-medium text-foreground">release channel</span> is the name your
              app asks for when it checks for updates. It is baked into the build and never changes.
            </p>
            <p className="mt-2">
              Mapping a channel to a branch decides which updates an app actually receives. Point{' '}
              <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs text-foreground">
                production
              </code>{' '}
              at a new branch to roll out, or back at the previous one to roll back, without
              shipping a new build. Or start a progressive rollout to serve a branch to a fraction
              of devices first.
            </p>
          </>
        }
      />
      {!!error && <ApiError error={error} />}

      <div className="space-y-4">
        {CONTROL_PLANE_ENABLED && !isAdmin && (
          <AdminOnlyNote>
            You are signed in with a member account, which is read-only. Ask an admin to create or
            delete channels, change which branch a channel serves, or manage progressive rollouts.
          </AdminOnlyNote>
        )}
        {CONTROL_PLANE_ENABLED && isAdmin && (
          <Card>
            <CardContent className="p-5">
              <form
                onSubmit={handleCreateChannel}
                className="grid items-end gap-4 sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
                <div className="space-y-1.5">
                  <Label htmlFor="channel-name">Channel name</Label>
                  <Input
                    id="channel-name"
                    placeholder="production, staging…"
                    value={newChannelName}
                    onChange={e => setNewChannelName(e.target.value)}
                    disabled={isCreating}
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>
                    Branch to serve{' '}
                    <span className="font-normal text-muted-foreground">(optional)</span>
                  </Label>
                  <SelectBranch
                    className="w-full"
                    currentBranch={newChannelBranch?.id ?? ''}
                    loading={isCreating}
                    onChange={(branchId, branchName) =>
                      setNewChannelBranch(
                        branchId && branchName ? { id: branchId, name: branchName } : null
                      )
                    }
                  />
                </div>
                <Button type="submit" disabled={isCreating || !newChannelName.trim()}>
                  <Plus className="h-4 w-4" />
                  {isCreating ? 'Creating…' : 'Create channel'}
                </Button>
              </form>
              <p className="mt-3 text-xs text-muted-foreground">
                Pick an existing branch or create one on the fly. You can also leave it empty and
                map a branch later from the table below.
              </p>
            </CardContent>
          </Card>
        )}

        <DataTable
          loading={isLoading}
          columns={tableColumns}
          data={data ?? []}
          emptyMessage="No channels yet. Create one to start serving updates to your builds."
        />
      </div>

      {CONTROL_PLANE_ENABLED && isAdmin && (
        <DeleteDialog
          isOpen={!!channelToDelete}
          onClose={() => setChannelToDelete(null)}
          onConfirm={handleExecuteDeletion}
          isDeleting={isDeleting}
          title="Delete channel"
          resourceName={channelToDelete?.releaseChannelName}
          descriptionText="Builds configured with this channel will stop receiving updates. This cannot be undone."
          confirmButtonText="Delete channel"
          isDeletingButtonText="Deleting…"
        />
      )}

      {CONTROL_PLANE_ENABLED && isAdmin && (
        <>
          <StartRolloutDialog
            channel={rolloutAction?.type === 'start' ? rolloutActionChannel : null}
            onClose={() => setRolloutAction(null)}
            onStarted={handleRolloutDone}
          />
          <ManageRolloutDialog
            channel={rolloutAction?.type === 'manage' ? rolloutActionChannel : null}
            onClose={() => setRolloutAction(null)}
            onDone={handleRolloutDone}
          />
        </>
      )}
    </div>
  );
};
