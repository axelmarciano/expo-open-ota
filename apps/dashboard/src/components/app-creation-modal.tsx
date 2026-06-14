import React, { useState } from 'react';
import { api, KeysMode, CreateAppPayload, ApiProblemError } from '@/lib/api';
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

type CreateAppModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onAppCreated?: (appId: string) => void;
};

export const CreateAppModal = ({
  isOpen,
  onClose,
  onAppCreated,
} : CreateAppModalProps) => {
  const { toast } = useToast();
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  const [name, setName] = useState('');
  const [keysMode, setKeysMode] = useState<KeysMode>('database');
  const [publicPath, setPublicPath] = useState('');
  const [privatePath, setPrivatePath] = useState('');
  const [publicSecretId, setPublicSecretId] = useState('');
  const [privateSecretId, setPrivateSecretId] = useState('');

  const resetForm = () => {
    setName('');
    setKeysMode('database');
    setPublicPath('');
    setPrivatePath('');
    setPublicSecretId('');
    setPrivateSecretId('');
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      toast({
        title: 'Validation Error',
        description: 'Please provide an application name.',
        variant: 'destructive',
      });
      return;
    }
    setIsSubmitting(true);
    const payload: CreateAppPayload = {
      name: name.trim(),
      keysConfig: {
        mode: keysMode,
        ...(keysMode === 'local' && {
          publicPath: publicPath.trim(),
          privatePath: privatePath.trim(),
        }),
        ...(keysMode === 'aws-secrets-manager' && {
          publicSecretId: publicSecretId.trim(),
          privateSecretId: privateSecretId.trim(),
        }),
      },
    };
    try {
      const response = await api.createApp(payload);
      toast({
        title: 'Success',
        description: `App "${name}" created successfully.`,
      });
      if (onAppCreated) {
        onAppCreated(response.appId);
      }
      handleClose();
    } catch (error) {
      let errorTitle = 'Error creating app';
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
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="sm:max-w-[480px]">
        <DialogHeader>
          <DialogTitle className="text-lg">Create New Application</DialogTitle>
          <DialogDescription className="text-sm">
            Register a new application target for OTA distributions and configure its isolated cryptographic signing keys.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-5 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="app-name" className="text-xs font-medium uppercase text-muted-foreground">
              Application Name
            </Label>
            <Input
              id="app-name"
              placeholder="e.g., consumer-mobile-app"
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={isSubmitting}
              className="h-9"
              required
            />
          </div>
          <div className="space-y-2">
            <Label className="text-xs font-medium uppercase text-muted-foreground">
              Key Management Mode
            </Label>
            <div className="grid grid-cols-1 gap-2">
              {[
                { id: 'database', label: 'Database Managed', desc: 'Backend natively manages storage and handling.' },
                { id: 'local', label: 'Local File Paths', desc: 'Bind references to local server key-pair structures.' },
                { id: 'aws-secrets-manager', label: 'AWS Secrets Manager', desc: 'Securely fetch keys directly from AWS vaults.' }
              ].map((mode) => (
                <label
                  key={mode.id}
                  className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
                    keysMode === mode.id
                      ? 'bg-accent/40 border-foreground/30 text-foreground'
                      : 'bg-background/50 border-border text-muted-foreground hover:bg-accent/20'
                  }`}
                >
                  <input
                    type="radio"
                    name="keysMode"
                    value={mode.id}
                    checked={keysMode === mode.id}
                    onChange={() => setKeysMode(mode.id as KeysMode)}
                    disabled={isSubmitting}
                    className="mt-0.5 accent-primary h-4 w-4"
                  />
                  <div className="flex flex-col gap-0.5">
                    <span className="text-sm font-medium text-foreground">{mode.label}</span>
                    <span className="text-xs text-muted-foreground">{mode.desc}</span>
                  </div>
                </label>
              ))}
            </div>
          </div>

          {keysMode === 'local' && (
            <div className="space-y-3 p-3 rounded-lg border border-dashed border-border bg-muted/20 animate-in fade-in-50 duration-200">
              <div className="space-y-1.5">
                <Label htmlFor="publicKeyPath" className="text-xs font-medium text-foreground">
                  Public Key Path
                </Label>
                <Input
                  id="publicKeyPath"
                  placeholder="/var/secrets/public.pem"
                  value={publicPath}
                  onChange={(e) => setPublicPath(e.target.value)}
                  disabled={isSubmitting}
                  className="h-9 bg-background"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="privateKeyPath" className="text-xs font-medium text-foreground">
                  Private Key Path
                </Label>
                <Input
                  id="privateKeyPath"
                  placeholder="/var/secrets/private.pem"
                  value={privatePath}
                  onChange={(e) => setPrivatePath(e.target.value)}
                  disabled={isSubmitting}
                  className="h-9 bg-background"
                  type="text"
                  required
                />
              </div>
            </div>
          )}

          {keysMode === 'aws-secrets-manager' && (
            <div className="space-y-3 p-3 rounded-lg border border-dashed border-border bg-muted/20 animate-in fade-in-50 duration-200">
              <div className="space-y-1.5">
                <Label htmlFor="publicSecretId" className="text-xs font-medium text-foreground">
                  AWS Secret ID (Public Key)
                </Label>
                <Input
                  id="publicSecretId"
                  placeholder="arn:aws:secretsmanager:..."
                  value={publicSecretId}
                  onChange={(e) => setPublicSecretId(e.target.value)}
                  disabled={isSubmitting}
                  className="h-9 bg-background"
                  required
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="privateSecretId" className="text-xs font-medium text-foreground">
                  AWS Secret ID (Private Key)
                </Label>
                <Input
                  id="privateSecretId"
                  placeholder="arn:aws:secretsmanager:..."
                  value={privateSecretId}
                  onChange={(e) => setPrivateSecretId(e.target.value)}
                  disabled={isSubmitting}
                  className="h-9 bg-background"
                  required
                />
              </div>
            </div>
          )}

          <DialogFooter className="pt-2 border-t border-border gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isSubmitting}
              className="h-9 text-xs font-medium"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isSubmitting}
              className="h-9 text-xs font-medium bg-foreground text-background hover:bg-foreground/90"
            >
              {isSubmitting ? 'Registering...' : 'Create Application'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
};