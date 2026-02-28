import { useMutation, useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { SelectBranch } from '@/pages/Channels/components/SelectBranch';
import { useCallback, useState } from 'react';
import { useToast } from '@/hooks/use-toast.ts';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Plus, Trash2 } from 'lucide-react';

export const Channels = () => {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: [`channels`],
    enabled: true,
    queryFn: () => api.getChannels(),
  });
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);

  // Create channel state
  const [createOpen, setCreateOpen] = useState(false);
  const [newChannelName, setNewChannelName] = useState('');

  // Delete channel state
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [channelToDelete, setChannelToDelete] = useState<string | null>(null);

  const updateMutation = useMutation({
    mutationKey: ['update-branch'],
    mutationFn: async ({
      branchName,
      releaseChannelId,
    }: {
      branchName: string;
      releaseChannelId: string;
    }) => {
      return api.updateChannelBranchMapping(branchName, {
        releaseChannel: releaseChannelId,
      });
    },
  });

  const createMutation = useMutation({
    mutationKey: ['create-channel'],
    mutationFn: (payload: { channelName: string }) => api.createChannel(payload),
  });

  const deleteMutation = useMutation({
    mutationKey: ['delete-channel'],
    mutationFn: (channelName: string) => api.deleteChannel(channelName),
  });

  const onBranchChange = useCallback(
    (channelId: string) => async (branchName?: string | null) => {
      if (!branchName) return;
      setLoading(true);
      try {
        await updateMutation.mutateAsync({
          branchName,
          releaseChannelId: channelId,
        });
        await refetch();
        toast({
          title: 'Branch updated',
          description: `Branch updated to ${branchName}`,
          duration: 2000,
        });
      } catch (error) {
        toast({
          title: 'Error updating branch',
          description: (error as { message: string }).message,
          variant: 'destructive',
        });
      } finally {
        setLoading(false);
      }
    },
    [updateMutation, toast, refetch]
  );

  const handleCreateChannel = async () => {
    if (!newChannelName.trim()) return;
    setLoading(true);
    try {
      await createMutation.mutateAsync({ channelName: newChannelName.trim() });
      await refetch();
      toast({
        title: 'Channel created',
        description: `Channel "${newChannelName.trim()}" has been created`,
        duration: 2000,
      });
      setNewChannelName('');
      setCreateOpen(false);
    } catch (error) {
      toast({
        title: 'Error creating channel',
        description: (error as { message: string }).message,
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteChannel = async () => {
    if (!channelToDelete) return;
    setLoading(true);
    try {
      await deleteMutation.mutateAsync(channelToDelete);
      await refetch();
      toast({
        title: 'Channel deleted',
        description: `Channel "${channelToDelete}" has been deleted`,
        duration: 2000,
      });
      setChannelToDelete(null);
      setDeleteOpen(false);
    } catch (error) {
      toast({
        title: 'Error deleting channel',
        description: (error as { message: string }).message,
        variant: 'destructive',
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="w-full h-screen flex-1 p-5">
      <div className="flex items-center justify-between mb-4">
        <h1 className="text-2xl font-medium">Channels</h1>
        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button size="sm">
              <Plus className="h-4 w-4 mr-1" />
              Create Channel
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create a new channel</DialogTitle>
              <DialogDescription>
                Enter a name for the new channel. You can associate a branch later.
              </DialogDescription>
            </DialogHeader>
            <Input
              placeholder="Channel name (e.g. production, staging)"
              value={newChannelName}
              onChange={e => setNewChannelName(e.target.value)}
              onKeyDown={e => {
                if (e.key === 'Enter') handleCreateChannel();
              }}
            />
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreateChannel} disabled={!newChannelName.trim() || loading}>
                Create
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>
      {!!error && <ApiError error={error} />}

      {/* Delete confirmation dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete channel</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the channel "{channelToDelete}"? This action cannot be
              undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDeleteChannel} disabled={loading}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <DataTable
        loading={isLoading}
        columns={[
          {
            header: 'Channel name',
            accessorKey: 'releaseChannelName',
            cell: value => {
              return (
                <span className="flex flex-row gap-2 items-center w-full">
                  {value.row.original.releaseChannelName}
                </span>
              );
            },
          },
          {
            header: 'Branch',
            accessorKey: 'releaseChannelName',
            cell: ({ row }) => {
              return (
                <SelectBranch
                  currentBranch={row.original.branchName || ''}
                  loading={isLoading || loading}
                  onChange={onBranchChange(row.original.releaseChannelName)}
                />
              );
            },
          },
          {
            header: '',
            id: 'actions',
            cell: ({ row }) => {
              return (
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={e => {
                    e.stopPropagation();
                    setChannelToDelete(row.original.releaseChannelName);
                    setDeleteOpen(true);
                  }}
                >
                  <Trash2 className="h-4 w-4 text-destructive" />
                </Button>
              );
            },
          },
        ]}
        data={data ?? []}
      />
    </div>
  );
};
