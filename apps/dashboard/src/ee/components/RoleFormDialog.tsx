// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

import { useEffect, useState } from 'react';
import { api, ApiProblemError, RoleRecord } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useToast } from '@/hooks/use-toast';
import { PERMISSION_GROUPS } from '@/ee/lib/permissionCatalog';

// Create/edit form for one role: a name and the permission catalog as
// toggles. `role` null means create; the dialog resets its draft every time
// it opens so a cancelled edit never leaks into the next one.
export const RoleFormDialog = ({
  open,
  role,
  onClose,
  onSaved,
}: {
  open: boolean;
  role: RoleRecord | null;
  onClose: () => void;
  onSaved: () => void;
}) => {
  const { toast } = useToast();
  const [name, setName] = useState('');
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setName(role?.name ?? '');
      setSelected(new Set(role?.permissions ?? []));
    }
  }, [open, role]);

  const togglePermission = (permission: string, next: boolean) => {
    setSelected(previous => {
      const draft = new Set(previous);
      if (next) {
        draft.add(permission);
      } else {
        draft.delete(permission);
      }
      return draft;
    });
  };

  const handleSave = async () => {
    if (!name.trim()) {
      toast({ title: 'Name required', description: 'Give the role a name.', variant: 'destructive' });
      return;
    }
    setIsSaving(true);
    try {
      const payload = { name: name.trim(), permissions: Array.from(selected) };
      if (role) {
        await api.updateRole(role.id, payload);
        toast({ title: 'Role updated', description: `"${payload.name}" was saved.` });
      } else {
        await api.createRole(payload);
        toast({ title: 'Role created', description: `"${payload.name}" is ready to assign.` });
      }
      onSaved();
      onClose();
    } catch (error) {
      let errorTitle = role ? 'Update failed' : 'Creation failed';
      let errorMessage = 'Could not save the role.';
      if (error instanceof ApiProblemError) {
        errorTitle = error.title;
        errorMessage = error.detail;
      } else if (error instanceof Error) {
        errorMessage = error.message;
      }
      toast({ title: errorTitle, description: errorMessage, variant: 'destructive' });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={isOpen => !isOpen && onClose()}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>{role ? 'Edit role' : 'New role'}</DialogTitle>
          <DialogDescription>
            A role is a reusable set of permissions. Assign it to a user on an app from the Users
            page.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-5">
          <div className="space-y-1.5">
            <Label htmlFor="role-name">Name</Label>
            <Input
              id="role-name"
              value={name}
              onChange={event => setName(event.target.value)}
              placeholder="e.g. Release manager"
              autoComplete="off"
            />
          </div>

          {PERMISSION_GROUPS.map(group => (
            <div key={group.label}>
              <p className="mb-2 text-sm font-medium">{group.label}</p>
              <div className="space-y-2.5 rounded-xl border p-3.5">
                {group.permissions.map(permission => (
                  <div key={permission.value} className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-sm">{permission.label}</p>
                      <p className="text-xs text-muted-foreground">{permission.description}</p>
                    </div>
                    <Switch
                      checked={selected.has(permission.value)}
                      onCheckedChange={next => togglePermission(permission.value, next)}
                      aria-label={permission.label}
                    />
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        <DialogFooter className="mt-2 gap-2 border-t pt-3 sm:gap-0">
          <Button type="button" variant="outline" onClick={onClose} disabled={isSaving}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} disabled={isSaving}>
            {isSaving ? 'Saving…' : role ? 'Save role' : 'Create role'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
