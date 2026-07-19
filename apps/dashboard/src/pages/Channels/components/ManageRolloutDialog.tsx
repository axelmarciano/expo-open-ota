import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Split, Undo2 } from 'lucide-react';
import { api, ChannelRecord, describeApiError } from '@/lib/api';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { PercentInput } from '@/components/rollout/PercentInput';

// Owns the edit / promote / revert mutations of an active channel rollout,
// plus the two confirmation dialogs. The body groups are self-contained so a
// future "Targeting rules" group can slot in above "Traffic split".
export const ManageRolloutDialog = ({
  channel,
  onClose,
  onDone,
}: {
  channel: ChannelRecord | null;
  onClose: () => void;
  onDone: () => void | Promise<void>;
}) => {
  const { toast } = useToast();
  const { selectedAppId } = useSelectedApp();
  const queryClient = useQueryClient();
  const rollout = channel?.rollout ?? null;
  const [percentage, setPercentage] = useState(rollout?.percentage ?? 10);
  const [confirm, setConfirm] = useState<'promote' | 'revert' | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [isEnding, setIsEnding] = useState(false);

  // Sync the input with the stored percentage each time a rollout opens.
  useEffect(() => {
    if (rollout) setPercentage(rollout.percentage);
  }, [rollout]);

  const busy = isSaving || isEnding;

  const handleSave = async () => {
    if (!channel) return;
    setIsSaving(true);
    try {
      await api.updateChannelRollout(channel.releaseChannelName, { percentage });
      toast({
        title: 'Rollout updated',
        description: `"${channel.releaseChannelName}" now rolls out to ${percentage}% of devices.`,
      });
      await onDone();
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not update the rollout');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setIsSaving(false);
    }
  };

  const handleEnd = async (outcome: 'promote' | 'revert') => {
    if (!channel || !rollout) return;
    setIsEnding(true);
    try {
      await api.endChannelRollout(channel.releaseChannelName, outcome);
      if (outcome === 'promote') {
        // The repoint changes every branch's release-channel linkage.
        queryClient.invalidateQueries({ queryKey: ['branches', selectedAppId] });
      }
      toast(
        outcome === 'promote'
          ? {
              title: 'Rollout promoted',
              description: `"${channel.releaseChannelName}" now serves "${rollout.rolloutBranchName}".`,
            }
          : {
              title: 'Rollout reverted',
              description: `"${channel.releaseChannelName}" is back on "${rollout.defaultBranchName}". Devices return after their next update check.`,
            }
      );
      await onDone();
      setConfirm(null);
      onClose();
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not end the rollout');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setIsEnding(false);
    }
  };

  return (
    <>
      <Dialog open={!!channel} onOpenChange={open => !open && !busy && onClose()}>
        <DialogContent className="sm:max-w-[480px]">
          <DialogHeader className="flex flex-col items-start gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-600">
              <Split className="h-5 w-5" />
            </div>
            <DialogTitle className="mt-2 text-lg font-semibold tracking-tight">
              Manage rollout
            </DialogTitle>
            <DialogDescription className="pt-1 text-left text-muted-foreground">
              <strong className="font-semibold text-foreground">
                "{rollout?.rolloutBranchName}"
              </strong>{' '}
              is rolling out to {rollout?.percentage}% of devices on{' '}
              <strong className="font-semibold text-foreground">
                "{channel?.releaseChannelName}"
              </strong>
              .
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-5 py-2">
            <div className="space-y-2">
              <Label>Traffic split</Label>
              <PercentInput
                value={percentage}
                onChange={setPercentage}
                min={1}
                max={99}
                disabled={busy}
              />
              <div className="flex justify-end">
                <Button
                  type="button"
                  size="sm"
                  onClick={handleSave}
                  disabled={busy || percentage === rollout?.percentage}
                  className="bg-emerald-600 text-white hover:bg-emerald-700">
                  {isSaving ? 'Saving…' : 'Save percentage'}
                </Button>
              </div>
            </div>

            <div className="space-y-2 border-t pt-4">
              <Label>End the rollout</Label>
              <button
                type="button"
                disabled={busy}
                onClick={() => setConfirm('promote')}
                className="flex w-full items-start gap-3 rounded-lg border px-3 py-3 text-left transition-colors hover:bg-muted/40 disabled:pointer-events-none disabled:opacity-50">
                <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-600">
                  <CheckCircle2 className="h-4 w-4" />
                </span>
                <span>
                  <span className="block text-sm font-medium">Promote</span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">
                    Every device switches to "{rollout?.rolloutBranchName}". It becomes the
                    channel's branch.
                  </span>
                </span>
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => setConfirm('revert')}
                className="flex w-full items-start gap-3 rounded-lg border px-3 py-3 text-left transition-colors hover:bg-muted/40 disabled:pointer-events-none disabled:opacity-50">
                <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-destructive/20 bg-destructive/10 text-destructive">
                  <Undo2 className="h-4 w-4" />
                </span>
                <span>
                  <span className="block text-sm font-medium">Revert</span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">
                    Every device returns to "{rollout?.defaultBranchName}". The rollout is
                    discarded.
                  </span>
                </span>
              </button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog
        open={confirm === 'promote'}
        onOpenChange={open => !open && !isEnding && setConfirm(null)}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader className="flex flex-col items-start gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-600">
              <CheckCircle2 className="h-5 w-5" />
            </div>
            <DialogTitle className="mt-2 text-lg font-semibold tracking-tight">
              Promote the rollout?
            </DialogTitle>
            <DialogDescription className="pt-1 text-left text-muted-foreground">
              Every device on{' '}
              <strong className="font-semibold text-foreground">
                "{channel?.releaseChannelName}"
              </strong>{' '}
              switches to{' '}
              <strong className="font-semibold text-foreground">
                "{rollout?.rolloutBranchName}"
              </strong>
              . It becomes the channel's mapped branch and the rollout ends.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4 gap-2 border-t pt-3 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setConfirm(null)}
              disabled={isEnding}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={() => handleEnd('promote')}
              disabled={isEnding}
              className="bg-emerald-600 text-white hover:bg-emerald-700">
              {isEnding ? 'Promoting…' : 'Promote'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={confirm === 'revert'}
        onOpenChange={open => !open && !isEnding && setConfirm(null)}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader className="flex flex-col items-start gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-destructive/10 border border-destructive/20 text-destructive">
              <AlertTriangle className="h-5 w-5" />
            </div>
            <DialogTitle className="mt-2 text-lg font-semibold tracking-tight">
              Revert the rollout?
            </DialogTitle>
            <DialogDescription className="pt-1 text-left text-muted-foreground">
              Every device on{' '}
              <strong className="font-semibold text-foreground">
                "{channel?.releaseChannelName}"
              </strong>{' '}
              returns to{' '}
              <strong className="font-semibold text-foreground">
                "{rollout?.defaultBranchName}"
              </strong>{' '}
              after their next update check. The rollout is discarded.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4 gap-2 border-t pt-3 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setConfirm(null)}
              disabled={isEnding}>
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => handleEnd('revert')}
              disabled={isEnding}>
              {isEnding ? 'Reverting…' : 'Revert'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
