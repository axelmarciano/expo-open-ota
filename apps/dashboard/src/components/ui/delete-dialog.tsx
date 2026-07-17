import { AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

type DeleteDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void | Promise<void>;
  isDeleting: boolean;
  title: string;
  resourceName: string | undefined;
  descriptionText: string;
  confirmButtonText?: string;
  isDeletingButtonText?: string;
};

export const DeleteDialog = ({
  isOpen,
  onClose,
  onConfirm,
  isDeleting,
  title,
  resourceName,
  descriptionText,
  confirmButtonText = 'Delete',
  isDeletingButtonText = 'Deleting...',
} : DeleteDialogProps) => {
  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader className="flex flex-col items-start gap-2">
          <div className="w-9 h-9 rounded-full bg-destructive/10 border border-destructive/20 flex items-center justify-center text-destructive">
            <AlertTriangle className="w-5 h-5" />
          </div>
          <DialogTitle className="text-lg font-semibold tracking-tight mt-2">
            {title}
          </DialogTitle>
          <DialogDescription className="pt-1 text-left text-muted-foreground">
            Are you completely sure you want to delete{' '}
            {resourceName && (
              <strong className="font-semibold text-foreground">
                "{resourceName}"
              </strong>
            )}
            ?
            <br /><br />
            {descriptionText}
          </DialogDescription>
        </DialogHeader>

        <DialogFooter className="pt-3 border-t gap-2 sm:gap-0 mt-4">
          <Button
            type="button"
            variant="outline"
            onClick={onClose}
            disabled={isDeleting}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={onConfirm}
            disabled={isDeleting}
          >
            {isDeleting ? isDeletingButtonText : confirmButtonText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};