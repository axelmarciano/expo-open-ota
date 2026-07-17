import { useSearchParams } from 'react-router';
import { useMemo } from 'react';
import { BranchesTable } from '@/pages/Updates/components/BranchesTable';
import { RuntimeVersionsTable } from '@/pages/Updates/components/RuntimeVersionsTable';
import { UpdatesTable } from '@/pages/Updates/components/UpdatesTable';
import { Separator } from '@/components/ui/separator';

export const Updates = () => {
  const [searchParams] = useSearchParams();
  const currentBranch = searchParams.get('branch');
  const runtimeVersion = searchParams.get('runtimeVersion');

  const component = useMemo(() => {
    if (!currentBranch) {
      return <BranchesTable />;
    }
    if (!runtimeVersion) {
      return <RuntimeVersionsTable branch={currentBranch} />;
    }
    return <UpdatesTable branch={currentBranch} runtimeVersion={runtimeVersion} />;
  }, [currentBranch, runtimeVersion]);

  return (
    <div className="w-full min-h-screen flex-1 p-5 space-y-6">
      <div className="space-y-2">
        <h1 className="text-2xl font-medium tracking-tight">Updates</h1>
        <p className="text-sm text-muted-foreground">
          Provision or delete release branches, explore targeted runtime environments, and audit deployed OTA bundle history.
        </p>
        <Separator />
      </div>
      {component}
    </div>
  );
};
