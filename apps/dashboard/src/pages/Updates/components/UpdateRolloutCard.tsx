import { useEffect, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Split } from 'lucide-react';
import { api, describeApiError, UpdateRolloutInfo } from '@/lib/api';
import { useSelectedApp } from '@/lib/SelectedAppContext';
import { useToast } from '@/hooks/use-toast';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { RolloutBar } from '@/components/rollout/RolloutBar';

// Renders the active per-update rollout for a (branch, runtime version) with
// its admin controls: progress forward, finish, or revert. `updates` holds one
// row per platform, all sharing the same update id and percentage.
export const UpdateRolloutCard = ({
  branch,
  runtimeVersion,
  updates,
  canManageRollout,
}: {
  branch: string;
  runtimeVersion: string;
  updates: UpdateRolloutInfo[];
  canManageRollout: boolean;
}) => {
  const { selectedAppId } = useSelectedApp();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const rollout = updates[0];
  const percentage = rollout.percentage;
  const updateId = rollout.updateId;

  const [nextPercentage, setNextPercentage] = useState(Math.min(99, percentage + 1));
  const [isBusy, setIsBusy] = useState(false);

  // The card is never remounted, so resync the input after an increase refreshes
  // the rollout percentage.
  useEffect(() => {
    setNextPercentage(Math.min(99, percentage + 1));
  }, [percentage, updateId]);

  const [confirm, setConfirm] = useState<'finish' | 'revert' | null>(null);

  const invalidate = (endsRollout: boolean) => {
    queryClient.invalidateQueries({ queryKey: ['updates', selectedAppId, branch, runtimeVersion] });
    queryClient.invalidateQueries({
      queryKey: ['update-rollout', selectedAppId, branch, runtimeVersion],
    });
    queryClient.invalidateQueries({ queryKey: ['update-details', selectedAppId] });
    if (endsRollout) {
      queryClient.invalidateQueries({ queryKey: ['runtimeVersions', selectedAppId, branch] });
    }
  };

  const isValidNextPercentage =
    Number.isInteger(nextPercentage) && nextPercentage > percentage && nextPercentage <= 99;

  const handleIncrease = async () => {
    if (!isValidNextPercentage) return;
    setIsBusy(true);
    try {
      await api.setUpdateRolloutPercentage(branch, runtimeVersion, {
        percentage: nextPercentage,
        expectedUpdateId: updateId,
      });
      toast({
        title: 'Rollout updated',
        description: `Update ${updateId} now rolls out to ${nextPercentage}% of devices.`,
      });
      invalidate(false);
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not update the rollout');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setIsBusy(false);
    }
  };

  const handleFinish = async () => {
    setIsBusy(true);
    try {
      await api.setUpdateRolloutPercentage(branch, runtimeVersion, {
        percentage: 100,
        expectedUpdateId: updateId,
      });
      toast({
        title: 'Rollout finished',
        description: `Update ${updateId} is now delivered to all devices.`,
      });
      invalidate(true);
      setConfirm(null);
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not finish the rollout');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setIsBusy(false);
    }
  };

  const handleRevert = async () => {
    setIsBusy(true);
    try {
      await api.revertUpdateRollout(branch, runtimeVersion, { expectedUpdateId: updateId });
      toast({
        title: 'Rollout reverted',
        description:
          'The previous update was republished. Devices return to it after their next update check.',
      });
      invalidate(true);
      setConfirm(null);
    } catch (error) {
      const { title, description } = describeApiError(error, 'Could not revert the rollout');
      toast({ title, description, variant: 'destructive' });
    } finally {
      setIsBusy(false);
    }
  };

  const canIncrease = percentage < 99;

  return (
    <>
      <Card className="mb-4 border-emerald-200 bg-emerald-50/40">
        <CardContent className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-1.5">
            <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700">
              <Split className="h-3.5 w-3.5" />
              Rollout in progress
            </span>
            <p className="text-xs text-muted-foreground">
              Update {updateId} · publishing to this branch and runtime version is paused until the
              rollout ends.
            </p>
            <RolloutBar value={percentage} />
          </div>

          {canManageRollout && (
            <div className="flex flex-wrap items-center gap-2">
              {canIncrease && (
                <div className="flex items-center gap-1.5">
                  <Input
                    type="number"
                    min={percentage + 1}
                    max={99}
                    value={nextPercentage}
                    disabled={isBusy}
                    onChange={e => setNextPercentage(Number(e.target.value))}
                    className="h-8 w-16"
                  />
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={isBusy || !isValidNextPercentage}
                    onClick={handleIncrease}>
                    Increase
                  </Button>
                </div>
              )}
              <Button
                type="button"
                size="sm"
                disabled={isBusy}
                onClick={() => setConfirm('finish')}
                className="bg-emerald-600 text-white hover:bg-emerald-700">
                Finish rollout
              </Button>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                disabled={isBusy}
                onClick={() => setConfirm('revert')}
                className="text-destructive hover:bg-destructive/10 hover:text-destructive">
                Revert
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={confirm === 'finish'}
        onOpenChange={open => !open && !isBusy && setConfirm(null)}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader className="flex flex-col items-start gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-600">
              <CheckCircle2 className="h-5 w-5" />
            </div>
            <DialogTitle className="mt-2 text-lg font-semibold tracking-tight">
              Finish the rollout?
            </DialogTitle>
            <DialogDescription className="pt-1 text-left text-muted-foreground">
              Update {updateId} will be delivered to all devices. Publishing to this branch and
              runtime version resumes.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4 gap-2 border-t pt-3 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setConfirm(null)}
              disabled={isBusy}>
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleFinish}
              disabled={isBusy}
              className="bg-emerald-600 text-white hover:bg-emerald-700">
              {isBusy ? 'Finishing…' : 'Finish rollout'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={confirm === 'revert'}
        onOpenChange={open => !open && !isBusy && setConfirm(null)}>
        <DialogContent className="sm:max-w-[420px]">
          <DialogHeader className="flex flex-col items-start gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-destructive/10 border border-destructive/20 text-destructive">
              <AlertTriangle className="h-5 w-5" />
            </div>
            <DialogTitle className="mt-2 text-lg font-semibold tracking-tight">
              Revert the rollout?
            </DialogTitle>
            <DialogDescription className="pt-1 text-left text-muted-foreground">
              The previous update will be republished as a new update, so every device returns to it
              after their next update check. Publishing resumes.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-4 gap-2 border-t pt-3 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => setConfirm(null)}
              disabled={isBusy}>
              Cancel
            </Button>
            <Button type="button" variant="destructive" onClick={handleRevert} disabled={isBusy}>
              {isBusy ? 'Reverting…' : 'Revert'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
