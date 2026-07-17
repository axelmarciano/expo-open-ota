import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { Badge } from '@/components/ui/badge.tsx';
import apple from '@/assets/apple.svg';
import android from '@/assets/android.svg';
import { UpdateDetailsRef, UpdateDetailsSheet } from '@/components/UpdateDetailsSheet';
import { useRef } from 'react';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { TimestampCell } from '@/components/ui/timestamp-cell';
import { UpdatesBreadcrumb } from '@/pages/Updates/components/UpdatesBreadcrumb';

export const UpdatesTable = ({
  branch,
  runtimeVersion,
}: {
  branch: string;
  runtimeVersion: string;
}) => {
  const sheetRef = useRef<UpdateDetailsRef>(null);
  const { selectedAppId } = useSelectedApp();
  const { data, isLoading, error } = useQuery({
    queryKey: ['updates', selectedAppId, branch, runtimeVersion],
    queryFn: () => api.getUpdates(branch, runtimeVersion),
    enabled: !!selectedAppId,
  });

  return (
    <div className="w-full flex-1">
      <UpdatesBreadcrumb branch={branch} runtimeVersion={runtimeVersion} />
      {!!error && <ApiError error={error} />}
      <UpdateDetailsSheet ref={sheetRef} branch={branch} runtimeVersion={runtimeVersion} />
      <DataTable
        loading={isLoading}
        columns={[
          {
            header: 'Update',
            accessorKey: 'updateId',
            cell: ({ row }) => (
              <span className="font-medium">{row.original.updateId}</span>
            ),
          },
          {
            header: 'UUID',
            accessorKey: 'updateUUID',
            cell: ({ row }) => (
              <span className="font-mono text-xs text-muted-foreground">
                {row.original.updateUUID}
              </span>
            ),
          },
          {
            header: 'Platform',
            accessorKey: 'platform',
            cell: ({ row }) => {
              const isIos = row.original.platform === 'ios';
              const isAndroid = row.original.platform === 'android';
              return (
                <div className="flex items-center gap-2">
                  {isIos && <img src={apple} className="w-4" alt="iOS" />}
                  {isAndroid && <img src={android} className="w-4" alt="Android" />}
                </div>
              );
            },
          },
          {
            header: 'Message',
            accessorKey: 'message',
            cell: ({ row }) => {
              const msg = row.original.message;
              return msg ? (
                <span className="block max-w-[200px] truncate text-sm text-muted-foreground">
                  {msg}
                </span>
              ) : (
                <span className="text-sm text-muted-foreground/60">—</span>
              );
            },
          },
          {
            header: 'Commit',
            accessorKey: 'commitHash',
            cell: ({ row }) => {
              return (
                <Badge variant="outline" className="font-mono text-xs">
                  {row.original.commitHash.slice(0, 7)}
                </Badge>
              );
            },
          },
          {
            header: 'Published',
            accessorKey: 'createdAt',
            cell: ({ row }) => (
              <TimestampCell dateString={row.original.createdAt} showSeconds />
            ),
          },
        ]}
        data={data ?? []}
        defaultSorting={[{ id: 'createdAt', desc: true }]}
        emptyMessage="No updates published for this runtime version yet."
        onRowClick={row => {
          sheetRef?.current?.openSheet(row);
        }}
      />
    </div>
  );
};
