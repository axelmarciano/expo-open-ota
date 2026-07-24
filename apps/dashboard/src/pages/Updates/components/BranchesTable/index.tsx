import { useMemo, useState } from 'react';
import { useMutation, useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import { Copy, GitBranch, Lock, Search, Trash2 } from 'lucide-react';
import { useNavigate } from 'react-router';
import { api, BranchRecord, describeApiError } from '@/lib/api';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useSettings } from '@/lib/SettingsContext';
import { useAppPermission } from '@/ee/lib/PermissionsContext';
import { useToast } from '@/hooks/use-toast';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { TimestampCell } from '@/components/ui/timestamp-cell';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { AdminOnlyNote } from '@/components/ui/admin-only-note';

type BranchSummary = BranchRecord & {
  latestRuntimeVersion?: string;
  latestUpdateAt?: string;
};

export const BranchesTable = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { selectedAppId } = useSelectedApp();
  const { CONTROL_PLANE_ENABLED } = useSettings();
  const canDeleteBranch = useAppPermission('branch:delete');
  const canProtectBranch = useAppPermission('branch:protect');
  const [search, setSearch] = useState('');
  const [branchToDelete, setBranchToDelete] = useState<BranchRecord | null>(null);
  const [branchBeingToggled, setBranchBeingToggled] = useState<string | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const branchesQuery = useQuery({
    queryKey: ['branches', selectedAppId],
    queryFn: () => api.getBranches(),
    enabled: !!selectedAppId,
  });
  const runtimeQueries = useQueries({
    queries: CONTROL_PLANE_ENABLED
      ? []
      : (branchesQuery.data ?? []).map(branch => ({
          queryKey: ['runtimeVersions', selectedAppId, branch.branchName],
          queryFn: () => api.getRuntimeVersions(branch.branchName),
          enabled: !!selectedAppId,
        })),
  });

  const summaries = useMemo<BranchSummary[]>(() => {
    const normalized = search.trim().toLowerCase();
    return (branchesQuery.data ?? [])
      .map((branch, index) => {
        const latestRuntime = runtimeQueries[index]?.data?.[0];
        return {
          ...branch,
          latestRuntimeVersion:
            branch.currentUpdate?.runtimeVersion ?? latestRuntime?.runtimeVersion,
          latestUpdateAt: branch.currentUpdate?.createdAt ?? latestRuntime?.lastUpdatedAt,
        };
      })
      .filter(branch => !normalized || branch.branchName.toLowerCase().includes(normalized));
  }, [branchesQuery.data, runtimeQueries, search]);

  const protectionMutation = useMutation({
    mutationFn: ({
      branchName,
      protected: isProtected,
    }: {
      branchName: string;
      protected: boolean;
    }) => api.setBranchProtection(branchName, isProtected),
  });

  const toggleProtection = async (branch: BranchRecord, next: boolean) => {
    setBranchBeingToggled(branch.branchName);
    try {
      await protectionMutation.mutateAsync({ branchName: branch.branchName, protected: next });
      await queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      toast({
        title: next ? 'Branch protected' : 'Branch unprotected',
        description: `"${branch.branchName}" is now ${next ? 'protected' : 'unprotected'}.`,
      });
    } catch (error) {
      const message = describeApiError(error, 'Could not update branch protection');
      toast({ title: message.title, description: message.description, variant: 'destructive' });
    } finally {
      setBranchBeingToggled(null);
    }
  };

  const deleteBranch = async () => {
    if (!branchToDelete) return;
    setIsDeleting(true);
    try {
      await api.deleteBranch(branchToDelete.branchName);
      await queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      toast({
        title: 'Branch deleted',
        description: `"${branchToDelete.branchName}" was removed.`,
      });
      setBranchToDelete(null);
    } catch (error) {
      const message = describeApiError(error, 'Could not delete branch');
      toast({ title: message.title, description: message.description, variant: 'destructive' });
    } finally {
      setIsDeleting(false);
    }
  };

  const columns = [
    {
      header: 'Branch',
      accessorKey: 'branchName',
      cell: ({ row }: { row: { original: BranchSummary } }) => (
        <span className="flex items-center gap-2.5 font-medium">
          <GitBranch className="h-4 w-4 text-muted-foreground" />
          {row.original.branchName}
          {row.original.protected && (
            <Lock className="h-3.5 w-3.5 text-emerald-700 dark:text-emerald-300" />
          )}
        </span>
      ),
    },
    {
      header: 'Latest runtime',
      accessorKey: 'latestRuntimeVersion',
      cell: ({ row }: { row: { original: BranchSummary } }) =>
        row.original.latestRuntimeVersion ? (
          <code className="font-mono text-xs">{row.original.latestRuntimeVersion}</code>
        ) : (
          <span className="text-muted-foreground/60">None</span>
        ),
    },
    {
      header: 'Latest update',
      accessorKey: 'latestUpdateAt',
      cell: ({ row }: { row: { original: BranchSummary } }) =>
        row.original.latestUpdateAt ? (
          <TimestampCell dateString={row.original.latestUpdateAt} />
        ) : (
          <span className="text-muted-foreground/60">None</span>
        ),
    },
    {
      header: 'ID',
      accessorKey: 'branchId',
      cell: ({ row }: { row: { original: BranchSummary } }) => (
        <div className="flex items-center gap-1">
          <code className="max-w-32 truncate font-mono text-xs text-muted-foreground">
            {row.original.branchId || 'Legacy'}
          </code>
          {row.original.branchId && (
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              title="Copy branch ID"
              onClick={event => {
                event.stopPropagation();
                void navigator.clipboard.writeText(row.original.branchId as string);
              }}>
              <Copy className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      ),
    },
    ...(CONTROL_PLANE_ENABLED && canProtectBranch
      ? [
          {
            header: 'Protected',
            id: 'protected',
            cell: ({ row }: { row: { original: BranchSummary } }) => (
              <div onClick={event => event.stopPropagation()}>
                <Switch
                  checked={row.original.protected}
                  disabled={branchBeingToggled === row.original.branchName}
                  onCheckedChange={next => void toggleProtection(row.original, next)}
                  aria-label={row.original.protected ? 'Unprotect branch' : 'Protect branch'}
                />
              </div>
            ),
          },
        ]
      : []),
    ...(CONTROL_PLANE_ENABLED && canDeleteBranch
      ? [
          {
            header: '',
            id: 'actions',
            cell: ({ row }: { row: { original: BranchSummary } }) => (
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                title={
                  row.original.protected
                    ? 'Unprotect this branch before deleting it'
                    : 'Delete branch'
                }
                disabled={row.original.protected}
                onClick={event => {
                  event.stopPropagation();
                  setBranchToDelete(row.original);
                }}>
                <Trash2 className="h-4 w-4" />
              </Button>
            ),
          },
        ]
      : []),
  ];

  return (
    <div className="space-y-4">
      {!!branchesQuery.error && <ApiError error={branchesQuery.error} />}
      {CONTROL_PLANE_ENABLED && !canDeleteBranch && !canProtectBranch && (
        <AdminOnlyNote>Branch management is read-only for your account.</AdminOnlyNote>
      )}
      <div className="relative max-w-sm">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          aria-label="Search branches"
          placeholder="Search branches"
          value={search}
          onChange={event => setSearch(event.target.value)}
          className="pl-9"
        />
      </div>
      <DataTable
        loading={
          branchesQuery.isLoading || runtimeQueries.some(runtimeQuery => runtimeQuery.isLoading)
        }
        columns={columns}
        data={summaries}
        emptyMessage={search ? 'No branches match this search.' : 'No branches yet.'}
        onRowClick={row => navigate(`/branches/${encodeURIComponent(row.branchName)}`)}
      />
      {CONTROL_PLANE_ENABLED && canDeleteBranch && (
        <DeleteDialog
          isOpen={!!branchToDelete}
          onClose={() => setBranchToDelete(null)}
          onConfirm={deleteBranch}
          isDeleting={isDeleting}
          title="Delete branch"
          resourceName={branchToDelete?.branchName}
          descriptionText="Updates on this branch will no longer be reachable from the dashboard. This cannot be undone."
          confirmButtonText="Delete branch"
          isDeletingButtonText="Deleting..."
        />
      )}
    </div>
  );
};
