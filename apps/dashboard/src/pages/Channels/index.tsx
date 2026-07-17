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
import { Separator } from '@/components/ui/separator';
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
      toast({ title: 'Channel Deleted', description: 'Release channel removed from application deployment grid.' });
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
        header: 'Channel Name',
        accessorKey: 'releaseChannelName',
        cell: ({ row }) => (
          <span className="flex flex-row gap-2 items-center w-full">
            {row.original.releaseChannelName}
          </span>
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
              header: 'Created At',
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
    <div className="w-full h-screen flex-1 p-5 space-y-6">
      <div className="space-y-2">
        <h1 className="text-2xl font-medium tracking-tight">Channels</h1>
        <p className="text-sm text-muted-foreground max-w-3xl">
          A <span className="font-medium text-foreground">branch</span> is a line of updates you
          publish to, much like a git branch. A{' '}
          <span className="font-medium text-foreground">release channel</span> is the name your app
          asks for when it checks for updates — it is baked into the build and never changes.
        </p>
        <p className="text-sm text-muted-foreground max-w-3xl">
          Mapping a channel to a branch is what decides which updates an app actually receives.
          Point{' '}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs text-foreground">
            production
          </code>{' '}
          at a new branch to roll out, or back at the previous one to roll back — without shipping a
          new build.
        </p>
        <Separator />
      </div>
      {!!error && <ApiError error={error} />}
      
      {CONTROL_PLANE_ENABLED && (
        <Card>
          <CardContent className="p-4 space-y-2">
            <form onSubmit={handleCreateChannel} className="flex gap-3 items-end flex-wrap">
              <div className="space-y-1.5">
                <Label
                  htmlFor="channel-name"
                  className="text-xs font-medium uppercase text-muted-foreground"
                >
                  New release channel
                </Label>
                <Input
                  id="channel-name"
                  placeholder="e.g., production, staging-v2"
                  value={newChannelName}
                  onChange={e => setNewChannelName(e.target.value)}
                  disabled={isCreating}
                  className="h-9 w-64"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium uppercase text-muted-foreground">
                  Branch to serve
                </Label>
                <SelectBranch
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
                <Plus className="mr-1.5 h-4 w-4" />
                {isCreating ? 'Creating...' : 'Create channel'}
              </Button>
            </form>
            <p className="text-xs text-muted-foreground">
              Pick an existing branch or create one on the fly. You can also leave it empty and map a
              branch later from the table below.
            </p>
          </CardContent>
        </Card>
      )}

      <DataTable
        loading={isLoading}
        columns={tableColumns}
        data={data ?? []}
      />

      {CONTROL_PLANE_ENABLED && (
        <DeleteDialog
          isOpen={!!channelToDelete}
          onClose={() => setChannelToDelete(null)}
          onConfirm={handleExecuteDeletion}
          isDeleting={isDeleting}
          title="Delete Release Channel"
          resourceName={channelToDelete?.releaseChannelName}
          descriptionText="Any runtime client devices polling for OTA updates against this track route string will no longer receive updates. This action is irreversible."
          confirmButtonText="Delete channel"
          isDeletingButtonText="Deleting..."
        />
      )}
    </div>
  );
};