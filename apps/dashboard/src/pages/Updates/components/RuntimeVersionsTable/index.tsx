import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { GitBranch, Milestone } from 'lucide-react';
import { useSearchParams } from 'react-router';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import { Badge } from '@/components/ui/badge.tsx';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { TimestampCell } from '@/components/ui/timestamp-cell';

export const RuntimeVersionsTable = ({ branch }: { branch: string }) => {
  const [, setSearchParams] = useSearchParams();
  const { selectedAppId } = useSelectedApp();
  const { data, isLoading, error } = useQuery({
    queryKey: ['runtimeVersions', selectedAppId, branch],
    queryFn: () => api.getRuntimeVersions(branch),
    enabled: !!selectedAppId,
  });

  return (
    <div className="w-full flex-1">
      <Breadcrumb className="mb-2">
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink href="/dashboard" className="flex items-center gap-2">
              <GitBranch className="w-4" />
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>{branch}</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>
      {!!error && <ApiError error={error} />}
      <DataTable
        loading={isLoading}
        columns={[
          {
            header: 'Runtime version',
            accessorKey: 'runtimeVersion',
            cell: value => {
              return (
                <button
                  className="flex flex-row gap-2 items-center cursor-pointer w-full underline"
                  onClick={() => {
                    setSearchParams({
                      branch,
                      runtimeVersion: value.row.original.runtimeVersion,
                    });
                  }}>
                  <Milestone className="w-4" />
                  {value.row.original.runtimeVersion}
                </button>
              );
            },
          },
          {
            header: 'Created at',
            accessorKey: 'createdAt',
            cell: ({ row }) => (
              <TimestampCell dateString={row.original.createdAt} showSeconds />
            ),
          },
          {
            header: 'Last update',
            accessorKey: 'lastUpdatedAt',
            cell: ({ row }) => (
              <TimestampCell dateString={row.original.lastUpdatedAt} showSeconds />
            ),
          },
          {
            header: '# Updates',
            accessorKey: 'numberOfUpdates',
            cell: ({ row }) => {
              return <Badge variant="secondary">{row.original.numberOfUpdates}</Badge>;
            },
          },
        ]}
        data={data ?? []}
        defaultSorting={[{ id: 'createdAt', desc: true }]}
      />
    </div>
  );
};
