'use client';

import * as React from 'react';
import { Check, ChevronsUpDown } from 'lucide-react';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';

// Sentinel for the pinned action row. It is not a selectable option, so it gets
// its own value and is excluded from search filtering.
const ACTION_VALUE = '__combobox_action__';

interface ComboboxProps {
  options: { value: string; label: string }[];
  value: string;
  onChange: (value: string) => void;
  loading?: boolean;
  label?: string;
  // Optional action pinned under the options (e.g. "New Application"). Stays
  // visible whatever the search input, since it is not one of the options.
  action?: { label: string; icon?: React.ReactNode; onSelect: () => void };
  // Extra classes for the trigger button — pass "w-full" to make the combobox
  // fill its container (the popover always matches the trigger width).
  className?: string;
}

export function Combobox(props: ComboboxProps) {
  const [open, setOpen] = React.useState(false);
  const { options, value, onChange, loading, label, action, className } = props;
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className={cn('w-max justify-between font-normal', className)}>
          <span className="truncate">
            {value ? options.find(opt => opt.value === value)?.label : label || 'Select option'}
          </span>
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[max(var(--radix-popover-trigger-width),12rem)] p-0">
        <Command
          filter={(itemValue, search) => {
            if (itemValue === ACTION_VALUE) return 1;
            const matchedOption = options.find(opt => opt.value === itemValue);
            const textToSearch = matchedOption ? matchedOption.label : itemValue;
            return textToSearch.toLowerCase().includes(search.toLowerCase()) ? 1 : 0;
          }}
        >
          <CommandInput placeholder="Search..." />
          <CommandList>
            <CommandEmpty>No option found.</CommandEmpty>
            <CommandGroup>
              {options.map(opt => (
                <CommandItem
                  key={opt.value}
                  value={opt.value}
                  onSelect={() => {
                    onChange(opt.value === value ? '' : opt.value);
                    setOpen(false);
                  }}>
                  <Check
                    className={cn(
                      'mr-2 h-4 w-4',
                      value === opt.value ? 'opacity-100' : 'opacity-0'
                    )}
                  />
                  {opt.label}
                </CommandItem>
              ))}
              {loading && <CommandItem disabled>Loading...</CommandItem>}
            </CommandGroup>
            {action && (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem
                    value={ACTION_VALUE}
                    onSelect={() => {
                      action.onSelect();
                      setOpen(false);
                    }}
                    className="text-gray-600">
                    {action.icon}
                    {action.label}
                  </CommandItem>
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
