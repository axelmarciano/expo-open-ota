import React from 'react';
import { Plus, LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

type ResourceCreateFormProps = {
  onSubmit: (e: React.FormEvent) => void | Promise<void>;
  inputValue: string;
  onInputChange: (value: string) => void;
  isSubmitting: boolean;
  label: string;
  placeholder: string;
  id?: string;
  buttonText?: string;
  icon?: LucideIcon;
};

// Compact inline creation row meant to sit right above the table it feeds:
// the placeholder carries the hint, the label is kept for screen readers.
export const ResourceCreateForm = ({
  onSubmit,
  inputValue,
  onInputChange,
  isSubmitting,
  label,
  placeholder,
  id = 'resource-name',
  buttonText = 'Create',
  icon: Icon = Plus,
}: ResourceCreateFormProps) => {
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || isSubmitting) return;
    onSubmit(e);
  };

  return (
    <form onSubmit={handleSubmit} className="flex items-center gap-2">
      <Input
        id={id}
        aria-label={label}
        placeholder={placeholder}
        value={inputValue}
        onChange={e => onInputChange(e.target.value)}
        disabled={isSubmitting}
        className="w-64"
      />
      <Button type="submit" disabled={isSubmitting || !inputValue.trim()} className="shrink-0">
        <Icon className="h-4 w-4" /> {isSubmitting ? 'Creating…' : buttonText}
      </Button>
    </form>
  );
};
