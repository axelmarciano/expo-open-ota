import { forwardRef, useImperativeHandle, useState } from 'react';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet.tsx';
import { Label } from '@/components/ui/label.tsx';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { Skeleton } from '@/components/ui/skeleton.tsx';
import { ApiError } from '@/components/APIError';
import { Badge } from '@/components/ui/badge.tsx';

interface Update {
  updateUUID: string;
  createdAt: string;
  updateId: string;
  platform: string;
  commitHash: string;
}

export type UpdateDetailsRef = {
  openSheet: (update: Update) => void;
  closeSheet: () => void;
};

const UpdateDetails = ({
  update,
  branch,
  runtimeVersion,
}: {
  update: Update | null;
  branch: string;
  runtimeVersion: string;
}) => {
  const { data, isLoading, error } = useQuery({
    queryKey: [`update-details-${update?.updateUUID}`],
    enabled: !!update?.updateId,
    queryFn: () => api.getUpdateDetails(branch, runtimeVersion, update?.updateId as string),
  });
  const updateDetails = data;
  if (!update) {
    return (
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Update details</SheetTitle>
        </SheetHeader>
        <Skeleton className="h-full w-full" />
      </SheetContent>
    );
  }
  if (isLoading) {
    return (
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Update details</SheetTitle>
          <SheetDescription>{update.updateId}</SheetDescription>
        </SheetHeader>
        <Skeleton className="h-full w-full" />
      </SheetContent>
    );
  }
  if (error) {
    return (
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Update details</SheetTitle>
          <SheetDescription>{update.updateId}</SheetDescription>
        </SheetHeader>
        <div className="flex flex-col items-center justify-center h-full">
          <ApiError error={error} />
        </div>
      </SheetContent>
    );
  }
  if (!updateDetails) {
    return (
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Update details</SheetTitle>
          <SheetDescription>{update.updateId}</SheetDescription>
        </SheetHeader>
        <Skeleton className="h-full w-full" />
      </SheetContent>
    );
  }
  return (
    <SheetContent style={{ maxWidth: 'none' }} className="w-[800px] overflow-scroll">
      <SheetHeader>
        <SheetTitle>Update details</SheetTitle>
        <SheetDescription>{updateDetails.updateId}</SheetDescription>
      </SheetHeader>
      <div className="grid gap-4 py-4 overflow-scroll">
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>Update ID</Label>
          <Badge variant="outline" className="col-span-3">
            {updateDetails.updateId}
          </Badge>
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>Branch</Label>
          <Badge variant="outline" className="col-span-3">
            {branch}
          </Badge>
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>Runtime version</Label>
          <Badge variant="outline" className="col-span-3">
            {runtimeVersion}
          </Badge>
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>Created At</Label>
          <Badge variant="outline" className="col-span-3">
            {new Date(updateDetails.createdAt).toLocaleDateString('en-GB', {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
              hour: 'numeric',
              minute: 'numeric',
              second: 'numeric',
            })}
          </Badge>
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>UUID</Label>
          <Badge variant="outline" className="col-span-3">
            {updateDetails.updateUUID}
          </Badge>
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>Commit</Label>
          <Badge variant="outline" className="col-span-3 break-all">
            {updateDetails.commitHash}
          </Badge>
        </div>
        {updateDetails.message && (
          <div className="grid grid-cols-4 items-center gap-4">
            <Label>Message</Label>
            <Badge variant="outline" className="col-span-3 break-all">
              {updateDetails.message}
            </Badge>
          </div>
        )}
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>Platform</Label>
          <Badge variant="outline" className="col-span-3 break-all">
            {updateDetails.platform}
          </Badge>
        </div>
        <div className="grid grid-cols-4 items-center gap-4">
          <Label>Type</Label>
          <Badge variant="outline" className="col-span-3 break-all">
            {updateDetails.type === 0 ? 'Normal update' : 'Rollback'}
          </Badge>
        </div>
      </div>
    </SheetContent>
  );
};

type Props = {
  branch: string;
  runtimeVersion: string;
};

export const UpdateDetailsSheet = forwardRef<UpdateDetailsRef, Props>(
  (
    {
      branch,
      runtimeVersion,
    }: {
      branch: string;
      runtimeVersion: string;
    },
    ref
  ) => {
    const [currentUpdate, setCurrentUpdate] = useState<Update | null>(null);
    useImperativeHandle(ref, () => ({
      openSheet: update => {
        setCurrentUpdate(update);
      },
      closeSheet: () => {
        setCurrentUpdate(null);
      },
    }));
    return (
      <Sheet
        open={!!currentUpdate}
        defaultOpen={false}
        onOpenChange={o => {
          if (!o) {
            setCurrentUpdate(null);
          }
        }}>
        <UpdateDetails branch={branch} runtimeVersion={runtimeVersion} update={currentUpdate} />
      </Sheet>
    );
  }
);
