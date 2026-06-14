import { useMutation, useQuery } from '@tanstack/react-query';
import { api, ChannelRecord, ApiProblemError } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { SelectBranch } from '@/pages/Channels/components/SelectBranch';
import { useCallback, useMemo, useState } from 'react';
import { useToast } from '@/hooks/use-toast.ts';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { Button } from '@/components/ui/button';
import { ResourceCreateForm } from '@/components/ui/resource-create-form';
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
  const [isCreating, setIsCreating] = useState(false);
  const [channelToDelete, setChannelToDelete] = useState<ChannelRecord | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const updateBranchMutation = useMutation({
    mutationKey: ['update-branch'],
    mutationFn: async ({
      branchName,
      releaseChannel,
    }: {
      branchName: string;
      releaseChannel: string;
    }) => {
      return api.updateChannelBranchMapping(branchName, {
        releaseChannel
      });
    },
  });

  const onBranchChange = useCallback(
    (channelId: string) => async (branchId?: string | null) => {
      if (!branchId) return;
      setLoading(true);
      try {
        await updateBranchMutation.mutateAsync({
          branchName: branchId,
          releaseChannel: channelId,
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
      await api.createChannel({ channelName: newChannelName.trim() });
      setNewChannelName('');
      await refetch();
      toast({ title: 'Channel Created', description: 'New release environment tracking route provisioned.' });
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
            onChange={onBranchChange(row.original.releaseChannelId)}
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
        <p className="text-sm text-muted-foreground">
          Manage distribution paths and map runtime bundle updates to client channels.
        </p>
        <Separator />
      </div>
      {!!error && <ApiError error={error} />}
      
      {CONTROL_PLANE_ENABLED && (
        <ResourceCreateForm
          id="channel-name"
          label="New Release Channel Name"
          placeholder="e.g., production, staging-v2"
          inputValue={newChannelName}
          onInputChange={setNewChannelName}
          onSubmit={handleCreateChannel}
          isSubmitting={isCreating}
          buttonText="Provision Channel"
          icon={Plus}
        />
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