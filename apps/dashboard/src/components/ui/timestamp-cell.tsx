import React from 'react';
import { Calendar } from 'lucide-react';
import { formatTimestamp } from '@/lib/utils';

type TimestampCellProps = {
  dateString: string | null | undefined;
  showSeconds?: boolean;
};

export const TimestampCell: React.FC<TimestampCellProps> = ({ 
  dateString, 
  showSeconds = false 
}) => {
  const formattedDate = formatTimestamp(dateString, showSeconds);

  return (
    <span className="flex items-center gap-1.5 text-gray-500 whitespace-nowrap">
      <Calendar className="w-3.5 h-3.5 text-gray-400" />
      {formattedDate ? (
        formattedDate
      ) : (
        <span className="text-gray-400 italic">Legacy Record</span>
      )}
    </span>
  );
};