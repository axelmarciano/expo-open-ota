import { Fragment } from 'react';
import { ChevronRight } from 'lucide-react';
import { useSearchParams } from 'react-router';

// Text breadcrumb for the Updates drill-down (branches → runtime versions →
// updates). Navigation happens through the same search params the page reads.
export const UpdatesBreadcrumb = ({
  branch,
  runtimeVersion,
}: {
  branch: string;
  runtimeVersion?: string;
}) => {
  const [, setSearchParams] = useSearchParams();

  const crumbs: { label: string; onClick?: () => void }[] = [
    { label: 'All branches', onClick: () => setSearchParams({}) },
    runtimeVersion
      ? { label: branch, onClick: () => setSearchParams({ branch }) }
      : { label: branch },
    ...(runtimeVersion ? [{ label: runtimeVersion }] : []),
  ];

  return (
    <nav aria-label="Breadcrumb" className="mb-4 flex items-center gap-1.5 text-sm">
      {crumbs.map((crumb, i) => (
        <Fragment key={`${crumb.label}-${i}`}>
          {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground/50" />}
          {crumb.onClick ? (
            <button
              onClick={crumb.onClick}
              className="rounded text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/30">
              {crumb.label}
            </button>
          ) : (
            <span className="font-medium text-foreground">{crumb.label}</span>
          )}
        </Fragment>
      ))}
    </nav>
  );
};
