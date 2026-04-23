import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { Box, GitBranch } from 'lucide-react';
import { useSearchParams } from 'react-router';
import { useSelectedApp } from '@/lib/SelectedAppContext';

export const BranchesTable = () => {
  const [, setSearchParams] = useSearchParams();
  const { selectedAppId } = useSelectedApp();
  const { data, isLoading, error } = useQuery({
    queryKey: ['branches', selectedAppId],
    queryFn: () => api.getBranches(),
    enabled: !!selectedAppId,
  });

  return (
    <div className="w-full flex-1">
      {!!error && <ApiError error={error} />}
      <DataTable
        loading={isLoading}
        columns={[
          {
            header: 'Branch name',
            accessorKey: 'branchName',
            cell: value => {
              return (
                <button
                  className="flex flex-row gap-2 items-center cursor-pointer w-full"
                  onClick={() => {
                    setSearchParams({ branch: value.row.original.branchName });
                  }}>
                  <GitBranch className="w-4" />
                  <span className="underline">{value.row.original.branchName}</span>
                </button>
              );
            },
          },
          {
            header: 'Release channel',
            size: 10,
            maxSize: 10,
            accessorKey: 'releaseChannel',
            cell: value => {
              const releaseChannel = value.row.original.releaseChannel;
              if (!releaseChannel) return <span>N/A</span>;
              return (
                <div className="flex flex-row gap-2 items-center">
                  <Box className="w-4" />
                  <span>{value.row.original.releaseChannel}</span>
                </div>
              );
            },
          },
        ]}
        data={data ?? []}
      />
    </div>
  );
};
