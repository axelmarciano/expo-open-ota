import type { BranchUpdateState } from '@/lib/api';

export const toBranchStatus = (state?: BranchUpdateState | null) => {
  if (!state) return undefined;
  return {
    label:
      state.rolloutPercentage != null ? ('Current rollout' as const) : ('Current update' as const),
    commitHash: state.commitHash.slice(0, 8),
    percentage: state.rolloutPercentage ?? undefined,
  };
};
