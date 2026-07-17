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
import { TimestampCell } from '@/components/ui/timestamp-cell';
import { Trash2, Plus } from 'lucide-react';
import { useSettings } from '@/lib/SettingsContext';

interface TableColumnConfig {
  header: string;
  accessorKey?: keyof ChannelRecord;
  id?: string;
  cell: (props: { row: { original: ChannelRecord } }) => React.ReactNode;
}

export const Channels = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
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
      toast({ title: 'Channel deleted', description: `"${channelToDelete.releaseChannelName}" was removed.` });
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
        cell: ({ row }) => (
          <span className="font-medium">{row.original.releaseChannelName}</span>
        ),
      },
      {
        header: 'Branch',
        accessorKey: 'branchId',
        cell: ({ row }) => (
          <SelectBranch
            currentBranch={row.original.branchId || ''}
            loading={isLoading || loading}
            onChange={onBranchChange(row.original)}
          />
        ),
      },
      ...(CONTROL_PLANE_ENABLED
        ? [
            {
              header: 'Created',
              accessorKey: 'createdAt',
              cell: ({ row }) => <TimestampCell dateString={row.original.createdAt} />,
            } satisfies TableColumnConfig,
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
                    title="Delete Release Channel"
                  >
                    <Trash2 />
                  </Button>
                </div>
              ),
            } satisfies TableColumnConfig,
          ]
        : []),
    ];
  }, [CONTROL_PLANE_ENABLED, isLoading, loading, onBranchChange]);

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
              app asks for when it checks for updates — it is baked into the build and never
              changes.
            </p>
            <p className="mt-2">
              Mapping a channel to a branch decides which updates an app actually receives. Point{' '}
              <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs text-foreground">
                production
              </code>{' '}
              at a new branch to roll out, or back at the previous one to roll back — without
              shipping a new build.
            </p>
          </>
        }
      />
      {!!error && <ApiError error={error} />}

      <div className="space-y-4">
        {CONTROL_PLANE_ENABLED && (
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

      {CONTROL_PLANE_ENABLED && (
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
    </div>
  );
};