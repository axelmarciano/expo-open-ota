import { useSearchParams } from 'react-router';
import { useMemo } from 'react';
import { BranchesTable } from '@/pages/Updates/components/BranchesTable';
import { RuntimeVersionsTable } from '@/pages/Updates/components/RuntimeVersionsTable';
import { UpdatesTable } from '@/pages/Updates/components/UpdatesTable';
import { PageHeader } from '@/components/PageHeader';

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
    <div className="w-full">
      <PageHeader
        title="Updates"
        description="Browse your release branches, drill into a runtime version, and audit every OTA update you have published."
      />
      {component}
    </div>
  );
};
