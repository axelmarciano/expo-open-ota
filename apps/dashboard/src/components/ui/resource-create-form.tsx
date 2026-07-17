import React from 'react';
import { Plus, LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent } from '@/components/ui/card';

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
} : ResourceCreateFormProps) => {
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || isSubmitting) return;
    onSubmit(e);
  };

  return (
    <Card>
      <CardContent className="p-4">
        <form onSubmit={handleSubmit} className="flex gap-3 items-end max-w-xl">
          <div className="flex-1 space-y-1">
            <Label htmlFor={id} className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              {label}
            </Label>
            <Input
              id={id}
              placeholder={placeholder}
              value={inputValue}
              onChange={(e) => onInputChange(e.target.value)}
              disabled={isSubmitting}
              className="bg-muted/30 focus-visible:bg-background"
            />
          </div>
          <Button 
            type="submit" 
            disabled={isSubmitting || !inputValue.trim()} 
            className="shrink-0"
          >
            <Icon className="mr-1" /> {isSubmitting ? 'Processing...' : buttonText}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
};