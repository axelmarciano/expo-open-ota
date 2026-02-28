import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { ApiError } from '@/components/APIError';
import { Combobox } from '@/components/Combobox';

export const SelectBranch = ({
  currentBranch,
  onChange,
  loading,
}: {
  onChange: (branchName?: string | null) => void;
  loading?: boolean;
  currentBranch?: string | null;
}) => {
  const { data, isLoading, error } = useQuery({
    queryKey: [`branches`],
    enabled: true,
    queryFn: () => api.getBranches(),
  });
  const allBranches =
    data
      ?.map(d => {
        return {
          branchName: d.branchName,
          id: d.branchName,
        };
      }) ?? [];
  if (error) {
    return <ApiError error={error} />;
  }
  return (
    <Combobox
      loading={isLoading || loading}
      options={allBranches.map(b => {
        return {
          label: b.branchName,
          value: b.id,
        };
      })}
      value={currentBranch || ''}
      onChange={onChange}
    />
  );
};
