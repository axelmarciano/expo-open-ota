import { useEffect, useState } from 'react';
import { Split } from 'lucide-react';
import { api, ChannelRecord, describeApiError } from '@/lib/api';
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
import { SelectBranch } from '@/pages/Channels/components/SelectBranch';
import { PercentInput } from '@/components/rollout/PercentInput';

// Owns the start-rollout mutation and its toast. Opened from the Channels
// table for a channel that has a branch mapped and no active rollout.
export const StartRolloutDialog = ({
  channel,
  onClose,
  onStarted,
}: {
  channel: ChannelRecord | null;
  onClose: () => void;
  onStarted: () => void | Promise<void>;
}) => {
  const { toast } = useToast();
  const [rolloutBranch, setRolloutBranch] = useState<{ id: string; name: string } | null>(null);
  const [percentage, setPercentage] = useState(10);
  const [isStarting, setIsStarting] = useState(false);

  // Reset the form each time a different channel opens the dialog.
  useEffect(() => {
    setRolloutBranch(null);
    setPercentage(10);
  }, [channel?.releaseChannelId]);

  const handleStart = async () => {
    if (!channel || !rolloutBranch) return;
    setIsStarting(true);
    try {
      await api.startChannelRollout(channel.releaseChannelName, {
        branchName: rolloutBranch.name,
        percentage,
      });
      toast({
        title: 'Rollout started',
        description: `${percentage}% of devices on "${channel.releaseChannelName}" now receive "${rolloutBranch.name}".`,
      });
      await onStarted();
      onClose();
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not start the rollout');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setIsStarting(false);
    }
  };

  return (
    <Dialog open={!!channel} onOpenChange={open => !open && !isStarting && onClose()}>
      <DialogContent className="sm:max-w-[460px]">
        <DialogHeader className="flex flex-col items-start gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-full border border-emerald-400/25 bg-emerald-400/10 text-emerald-700 dark:text-emerald-300">
            <Split className="h-5 w-5" />
          </div>
          <DialogTitle className="mt-2 text-lg font-semibold tracking-tight">
            Start a rollout
          </DialogTitle>
          <DialogDescription className="pt-1 text-left text-muted-foreground">
            Serve a rollout branch to a fraction of devices on{' '}
            <strong className="font-semibold text-foreground">
              "{channel?.releaseChannelName}"
            </strong>
            . The rest keep receiving{' '}
            <strong className="font-semibold text-foreground">"{channel?.branchName}"</strong>.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-5 py-2">
          <div className="space-y-1.5">
            <Label>Rollout branch</Label>
            <SelectBranch
              className="w-full"
              currentBranch={rolloutBranch?.id ?? ''}
              excludeBranchIds={channel?.branchId ? [channel.branchId] : []}
              onChange={(branchId, branchName) =>
                setRolloutBranch(branchId && branchName ? { id: branchId, name: branchName } : null)
              }
            />
            <p className="text-xs text-muted-foreground">
              Devices in the rollout receive updates from this branch instead of the mapped one.
            </p>
          </div>

          <div className="space-y-1.5">
            <Label>Traffic split</Label>
            <PercentInput value={percentage} onChange={setPercentage} min={1} max={99} />
          </div>
        </div>

        <DialogFooter className="mt-2 gap-2 border-t pt-3 sm:gap-0">
          <Button type="button" variant="outline" onClick={onClose} disabled={isStarting}>
            Cancel
          </Button>
          <Button type="button" onClick={handleStart} disabled={isStarting || !rolloutBranch}>
            {isStarting ? 'Starting…' : 'Start rollout'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
