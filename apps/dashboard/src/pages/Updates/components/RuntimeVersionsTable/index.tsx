import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { Milestone } from 'lucide-react';
import { Badge } from '@/components/ui/badge.tsx';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { TimestampCell } from '@/components/ui/timestamp-cell';

export const RuntimeVersionsTable = ({
  branch,
  onSelectRuntime,
}: {
  branch: string;
  onSelectRuntime: (runtimeVersion: string) => void;
}) => {
  const { selectedAppId } = useSelectedApp();
  const { data, isLoading, error } = useQuery({
    queryKey: ['runtimeVersions', selectedAppId, branch],
    queryFn: () => api.getRuntimeVersions(branch),
    enabled: !!selectedAppId,
  });

  return (
    <div className="w-full flex-1">
      {!!error && <ApiError error={error} />}
      <DataTable
        loading={isLoading}
        columns={[
          {
            header: 'Runtime version',
            accessorKey: 'runtimeVersion',
            cell: ({ row }) => (
              <span className="flex items-center gap-2.5">
                <span className="flex h-7 w-7 items-center justify-center rounded-md bg-muted text-muted-foreground">
                  <Milestone className="h-3.5 w-3.5" />
                </span>
                <span className="font-medium">{row.original.runtimeVersion}</span>
                {(row.original.rolloutPercentage != null || row.original.activeRollout) && (
                  <Badge className="border-emerald-400/25 bg-emerald-400/10 text-emerald-700 dark:text-emerald-300">
                    {row.original.rolloutPercentage != null
                      ? `${row.original.rolloutPercentage}% rollout`
                      : 'Rollout'}
                  </Badge>
                )}
              </span>
            ),
          },
          {
            header: 'Created',
            accessorKey: 'createdAt',
            cell: ({ row }) => <TimestampCell dateString={row.original.createdAt} showSeconds />,
          },
          {
            header: 'Last update',
            accessorKey: 'lastUpdatedAt',
            cell: ({ row }) => (
              <TimestampCell dateString={row.original.lastUpdatedAt} showSeconds />
            ),
          },
          {
            header: 'Updates',
            accessorKey: 'numberOfUpdates',
            cell: ({ row }) => {
              return <Badge variant="secondary">{row.original.numberOfUpdates}</Badge>;
            },
          },
        ]}
        data={data ?? []}
        defaultSorting={[{ id: 'createdAt', desc: true }]}
        emptyMessage="No runtime versions on this branch yet."
        onRowClick={row => onSelectRuntime(row.runtimeVersion)}
      />
    </div>
  );
};
