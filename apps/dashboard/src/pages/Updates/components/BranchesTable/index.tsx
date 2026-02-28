import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { Box, GitBranch } from 'lucide-react';
import { useSearchParams } from 'react-router';

export const BranchesTable = () => {
  const [, setSearchParams] = useSearchParams();
  const { data, isLoading, error } = useQuery({
    queryKey: ['branches'],
    queryFn: () => api.getBranches(),
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
            header: 'Release channel(s)',
            size: 10,
            maxSize: 10,
            accessorKey: 'releaseChannels',
            cell: value => {
              const releaseChannels = value.row.original.releaseChannels;
              if (!releaseChannels?.length) return <span>N/A</span>;
              return (
                releaseChannels.map(channel => (
                  <div key={channel} className="flex flex-row gap-2 items-center">
                    <Box className="w-4" />
                    <span>{channel}</span>
                  </div>
                ))
              )
            },
          },
        ]}
        data={data ?? []}
      />
    </div>
  );
};
