import { useMutation, useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { SelectBranch } from '@/pages/Channels/components/SelectBranch';
import { useCallback, useState } from 'react';
import { useToast } from '@/hooks/use-toast.ts';
import { useSelectedApp } from '@/lib/SelectedAppContext';

export const Channels = () => {
  const { selectedAppId } = useSelectedApp();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['channels', selectedAppId],
    enabled: !!selectedAppId,
    queryFn: () => api.getChannels(),
  });
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);

  const mutation = useMutation({
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
    (channelId: string) => async (branchName?: string | null) => {
      if (!branchName) return;
      setLoading(true);
      try {
        await mutation.mutateAsync({
          branchName,
          releaseChannel: channelId,
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
    [mutation, toast]
  );
  return (
    <div className="w-full h-screen flex-1 p-5">
      <h1 className="text-2xl font-medium mb-4">Channels</h1>
      {!!error && <ApiError error={error} />}
      <DataTable
        loading={isLoading}
        columns={[
          {
            header: 'Channel name',
            accessorKey: 'releaseChannelId',
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
              console.log(row);
              return (
                <SelectBranch
                  currentBranch={row.original.branchId || ''}
                  loading={isLoading || loading}
                  onChange={onBranchChange(row.original.releaseChannelId)}
                />
              );
            },
          },
        ]}
        data={data ?? []}
      />
    </div>
  );
};
