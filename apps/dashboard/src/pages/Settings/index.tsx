import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api.ts';
import { DataTable } from '@/components/DataTable';
import { ApiError } from '@/components/APIError';

// formatSettingValue renders any /api/settings field as a plain string so the
// DataTable's default cell renderer works without a per-row custom cell.
// Strings pass through; APPS (array of {id, name}) is flattened to a human-
// readable "Name (id)" list so the user sees both fields at a glance.
function formatSettingValue(value: unknown): string {
  if (typeof value === 'string') return value;
  if (Array.isArray(value)) {
    return value
      .map(v => {
        if (v && typeof v === 'object' && 'id' in v) {
          const { id, name } = v as { id: string; name?: string };
          return name ? `${name} (${id})` : id;
        }
        return String(v);
      })
      .join(', ');
  }
  if (value == null) return '';
  return String(value);
}

export const Settings = () => {
  const { data, isLoading, error } = useQuery({
    queryKey: ['settings'],
    queryFn: () => api.getSettings(),
  });
  return (
    <div className="w-full h-screen flex-1 p-5">
      <h1 className="text-2xl font-medium mb-4">Settings</h1>
      {!!error && <ApiError error={error} />}
      <DataTable
        columns={[
          {
            header: 'Key',
            accessorKey: 'key',
          },
          {
            header: 'Value',
            accessorKey: 'value',
          },
        ]}
        data={Object.entries(data || {}).map(([key, value]) => ({
          key,
          value: formatSettingValue(value),
        }))}
        loading={isLoading}
      />
    </div>
  );
};
