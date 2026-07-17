import React, { useState } from 'react';
import { api, describeApiError } from '@/lib/api';
import { useToast } from '@/hooks/use-toast';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { PasswordRulesChecklist } from '@/components/ui/password-rules-checklist';
import { isPasswordValid } from '@/lib/password-policy';

type CreateUserModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onUserCreated?: () => void;
};

export const CreateUserModal = ({ isOpen, onClose, onUserCreated }: CreateUserModalProps) => {
  const { toast } = useToast();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isAdmin, setIsAdmin] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleClose = () => {
    setEmail('');
    setPassword('');
    setIsAdmin(false);
    onClose();
  };

  const canSubmit = email.trim().length > 0 && isPasswordValid(password);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit || isSubmitting) return;
    setIsSubmitting(true);
    try {
      const user = await api.createUser({ email: email.trim(), password, isAdmin });
      toast({
        title: 'User created',
        description: `"${user.email}" can now sign in to the dashboard.`,
      });
      onUserCreated?.();
      handleClose();
    } catch (error) {
      toast({ ...describeApiError(error, 'Error creating user'), variant: 'destructive' });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={open => !open && handleClose()}>
      <DialogContent className="sm:max-w-[420px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle className="text-lg">Create user</DialogTitle>
            <DialogDescription>
              The user signs in with this email and password. Admins can additionally manage users,
              create apps and remap release channels.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-1.5">
              <Label htmlFor="new-user-email" className="text-xs font-medium text-foreground">
                Email
              </Label>
              <Input
                id="new-user-email"
                type="email"
                placeholder="e.g., jane@acme.dev"
                value={email}
                onChange={e => setEmail(e.target.value)}
                disabled={isSubmitting}
                autoFocus
                className="h-9"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="new-user-password" className="text-xs font-medium text-foreground">
                Password
              </Label>
              <Input
                id="new-user-password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={e => setPassword(e.target.value)}
                disabled={isSubmitting}
                className="h-9"
              />
              <PasswordRulesChecklist password={password} />
            </div>
            <label className="flex items-center gap-2 text-sm" htmlFor="new-user-admin">
              <input
                id="new-user-admin"
                type="checkbox"
                checked={isAdmin}
                onChange={e => setIsAdmin(e.target.checked)}
                disabled={isSubmitting}
                className="h-4 w-4 rounded border-input accent-primary"
              />
              <span>Admin account</span>
            </label>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={isSubmitting}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting || !canSubmit}>
              {isSubmitting ? 'Creating...' : 'Create user'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};
