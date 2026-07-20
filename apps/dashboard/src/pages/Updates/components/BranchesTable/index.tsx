import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiProblemError, BranchRecord, describeApiError } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { DataTable } from '@/components/DataTable';
import { GitBranch, Lock, Plus, ShieldAlert, Trash2 } from 'lucide-react';
import { useSearchParams } from 'react-router';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useCallback, useMemo, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useToast } from '@/hooks/use-toast.ts';
import { ResourceCreateForm } from '@/components/ui/resource-create-form';
import { DeleteDialog } from '@/components/ui/delete-dialog';
import { AdminOnlyNote } from '@/components/ui/admin-only-note';
import { useSettings } from '@/lib/SettingsContext';
import { useAppPermission } from '@/ee/lib/PermissionsContext';
import { EnterpriseExplainerDialog } from '@/ee/components/EnterpriseExplainerDialog';

interface TableColumnConfig {
  header: string;
  accessorKey?: keyof BranchRecord;
  id?: string;
  size?: number;
  maxSize?: number;
  cell: (props: { row: { original: BranchRecord } }) => React.ReactNode;
}

export const BranchesTable = () => {
  const { CONTROL_PLANE_ENABLED } = useSettings();
  // Display gating only: the server re-checks each of these permissions on
  // its route (admins always pass, members follow their enterprise grants).
  const canCreateBranch = useAppPermission('branch:create');
  const canDeleteBranch = useAppPermission('branch:delete');
  const canProtectBranch = useAppPermission('branch:protect');
  const [, setSearchParams] = useSearchParams();
  const { selectedAppId } = useSelectedApp();
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const [newBranchName, setNewBranchName] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const [branchToDelete, setBranchToDelete] = useState<BranchRecord | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  // Branch protection is enterprise: without a valid license the toggle opens
  // the explainer dialog instead of calling the API.
  const [isExplainerOpen, setIsExplainerOpen] = useState(false);
  const [branchBeingToggled, setBranchBeingToggled] = useState<string | null>(null);
  // Protecting a branch can lock out CI tokens, so it goes through a
  // confirmation; unprotecting is immediate.
  const [branchToProtect, setBranchToProtect] = useState<BranchRecord | null>(null);
  const [isProtecting, setIsProtecting] = useState(false);

  const licenseQuery = useQuery({
    queryKey: ['license'],
    queryFn: () => api.getLicense(),
    enabled: CONTROL_PLANE_ENABLED,
  });
  const isEnterprise = !!licenseQuery.data?.valid;

  const { data, isLoading, error } = useQuery({
    queryKey: ['branches', selectedAppId],
    queryFn: () => api.getBranches(),
    enabled: !!selectedAppId,
  });

   const applyProtection = useCallback(async (branch: BranchRecord, nextProtected: boolean) => {
    setBranchBeingToggled(branch.branchName);
    setIsProtecting(true);
    try {
      await api.setBranchProtection(branch.branchName, nextProtected);
      queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      toast({
        title: nextProtected ? 'Branch protected' : 'Branch unprotected',
        description: nextProtected
          ? `Only tokens allowed on protected branches can publish to "${branch.branchName}" now.`
          : `Any token can publish to "${branch.branchName}" again.`,
      });
      setBranchToProtect(null);
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not update branch protection');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setBranchBeingToggled(null);
      setIsProtecting(false);
    }
  }, [queryClient, selectedAppId, toast]);

  const handleToggleProtection = useCallback((branch: BranchRecord, nextProtected: boolean) => {
    if (!isEnterprise) {
      setIsExplainerOpen(true);
      return;
    }
    if (nextProtected) {
      setBranchToProtect(branch);
      return;
    }
    applyProtection(branch, false);
  }, [isEnterprise, applyProtection]);

 

  const handleCreateBranch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newBranchName.trim()) return;

    setIsCreating(true);
    try {
      await api.createBranch(newBranchName.trim());
      setNewBranchName('');
      queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      toast({
        title: 'Branch created',
        description: `"${newBranchName.trim()}" is ready to receive updates.`,
      });
    } catch (error) {
      let errorTitle = 'Error creating branch';
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
    if (!branchToDelete) return;
    setIsDeleting(true);
    try {
      await api.deleteBranch(branchToDelete.branchName);
      queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      toast({
        title: 'Branch deleted',
        description: `"${branchToDelete.branchName}" and its updates were removed.`,
      });
      setBranchToDelete(null);
    } catch (error) {
      let errorTitle = 'Deletion failed';
      let errorMessage = 'Could not delete the branch.';
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
        header: 'Branch',
        accessorKey: 'branchName',
        cell: ({ row }) => (
          <span className="flex items-center gap-2.5">
            <span className="flex h-7 w-7 items-center justify-center rounded-md bg-muted text-muted-foreground">
              <GitBranch className="h-3.5 w-3.5" />
            </span>
            <span className="font-medium">{row.original.branchName}</span>
          </span>
        ),
      },
      {
        header: 'Release channel',
        accessorKey: 'releaseChannel',
        cell: ({ row }) => {
          const releaseChannel = row.original.releaseChannel;
          if (!releaseChannel) {
            return <span className="text-muted-foreground/60">Not mapped</span>;
          }
          return <Badge variant="outline">{releaseChannel}</Badge>;
        },
      },
      ...(CONTROL_PLANE_ENABLED
        ? [
            {
              header: 'Protection',
              id: 'protection',
              cell: ({ row }) => {
                const isProtected = row.original.protected;
                const label = (
                  <span
                    className={
                      isProtected
                        ? 'inline-flex items-center gap-1 text-sm font-medium text-emerald-700'
                        : 'text-sm text-muted-foreground'
                    }
                  >
                    {isProtected && <Lock className="h-3.5 w-3.5" />}
                    {isProtected ? 'Protected' : 'Unprotected'}
                  </span>
                );
                // Without the permission the state is read-only; the server
                // gates the write anyway.
                if (!canProtectBranch) {
                  return label;
                }
                return (
                  <div className="flex items-center gap-2.5" onClick={e => e.stopPropagation()}>
                    <Switch
                      checked={isProtected}
                      disabled={branchBeingToggled === row.original.branchName}
                      onCheckedChange={next => handleToggleProtection(row.original, next)}
                      aria-label={isProtected ? 'Unprotect branch' : 'Protect branch'}
                    />
                    {label}
                  </div>
                );
              },
            } satisfies TableColumnConfig,
          ]
        : []),
      ...(CONTROL_PLANE_ENABLED && canDeleteBranch
        ? [
            {
              header: '',
              id: 'actions',
              cell: ({ row }) => {
                const isProtected = row.original.protected;
                return (
                  <div
                    className="text-right"
                    title={
                      isProtected
                        ? 'Protected branches cannot be deleted. Remove the protection first.'
                        : undefined
                    }
                  >
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={e => {
                        e.stopPropagation();
                        setBranchToDelete(row.original);
                      }}
                      disabled={isProtected}
                      className="h-8 w-8 p-0 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                      title={isProtected ? undefined : 'Delete branch'}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                );
              },
            } satisfies TableColumnConfig,
          ]
        : []),
    ];
  }, [
    CONTROL_PLANE_ENABLED,
    canDeleteBranch,
    canProtectBranch,
    branchBeingToggled,
    handleToggleProtection,
  ]);

  return (
    <div className="w-full flex-1 space-y-4">
      {!!error && <ApiError error={error} />}

      {CONTROL_PLANE_ENABLED && !canCreateBranch && !canDeleteBranch && !canProtectBranch && (
        <AdminOnlyNote>
          You do not have permission to manage branches on this app. Ask an admin to grant you
          access.
        </AdminOnlyNote>
      )}
      {CONTROL_PLANE_ENABLED && canCreateBranch && (
        <div className="flex justify-end">
          <ResourceCreateForm
            id="branch-name"
            label="New branch name"
            placeholder="New branch name, e.g. main"
            inputValue={newBranchName}
            onInputChange={setNewBranchName}
            onSubmit={() => handleCreateBranch({ preventDefault: () => {} } as React.FormEvent)}
            isSubmitting={isCreating}
            buttonText="Create branch"
            icon={Plus}
          />
        </div>
      )}

      <DataTable
        loading={isLoading}
        columns={tableColumns}
        data={data ?? []}
        emptyMessage="No branches yet. Publish an update or create a branch to get started."
        onRowClick={row => setSearchParams({ branch: row.branchName })}
      />

      {CONTROL_PLANE_ENABLED && canDeleteBranch && (
        <DeleteDialog
          isOpen={!!branchToDelete}
          onClose={() => setBranchToDelete(null)}
          onConfirm={handleExecuteDeletion}
          isDeleting={isDeleting}
          title="Delete branch"
          resourceName={branchToDelete?.branchName}
          descriptionText="Every update published to this branch will be permanently deleted. Apps pointing at a channel mapped to this branch will stop receiving updates."
          confirmButtonText="Delete branch"
          isDeletingButtonText="Deleting…"
        />
      )}

      <Dialog
        open={!!branchToProtect}
        onOpenChange={open => !open && !isProtecting && setBranchToProtect(null)}
      >
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader className="flex flex-col items-start gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-600">
              <ShieldAlert className="h-5 w-5" />
            </div>
            <DialogTitle className="mt-2 text-lg font-semibold tracking-tight">
              Protect this branch?
            </DialogTitle>
            <DialogDescription className="pt-1 text-left text-muted-foreground">
              Once{' '}
              <strong className="font-semibold text-foreground">
                "{branchToProtect?.branchName}"
              </strong>{' '}
              is protected, only API tokens explicitly allowed on protected branches can publish,
              roll back or republish on it. Tokens handed to developers will be blocked, and the
              branch cannot be deleted until the protection is lifted.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4 gap-2 border-t pt-3 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setBranchToProtect(null)}
              disabled={isProtecting}
            >
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => branchToProtect && applyProtection(branchToProtect, true)}
              disabled={isProtecting}
              className="bg-emerald-600 text-white hover:bg-emerald-700"
            >
              {isProtecting ? 'Protecting…' : 'Protect branch'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <EnterpriseExplainerDialog
        open={isExplainerOpen}
        onOpenChange={setIsExplainerOpen}
        feature={{
          name: 'Branch protection',
          description:
            'Protect critical branches like production. Once a branch is protected, only API tokens you explicitly allow can publish, roll back or republish on it, so a token handed to a developer for staging can never ship to production.',
        }}
      />
    </div>
  );
};
