import { Input } from 'antd';
import type { ReactNode } from 'react';
import { Controller, useFormFields } from './form';
import { Field } from './field';

// A text field, a password field, one component: the difference is a prop, because the difference at
// the API is a prop too.

function messageOf(errors: Record<string, unknown>, name: string): string | undefined {
  const found = errors[name];
  if (found && typeof found === 'object' && 'message' in found) {
    return String((found as { message?: unknown }).message ?? '');
  }
  return undefined;
}

export type TextInputProps = {
  name: string;
  label: string;
  hint?: string;
  icon?: ReactNode;
  fullWidth?: boolean;
  password?: boolean;
  placeholder?: string;
  autoFocus?: boolean;
  autoComplete?: string;
};

export function TextInput({
  name,
  label,
  hint,
  icon,
  fullWidth = true,
  password,
  placeholder,
  autoFocus,
  autoComplete,
}: TextInputProps) {
  const { control, errors } = useFormFields();
  const error = messageOf(errors as Record<string, unknown>, name);
  const Input_ = password ? Input.Password : Input;

  return (
    <Field label={label} htmlFor={name} error={error} hint={hint} fullWidth={fullWidth}>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Input_
            {...field}
            id={name}
            value={(field.value as string) ?? ''}
            onChange={(event) => field.onChange(event.target.value)}
            prefix={icon}
            placeholder={placeholder}
            autoFocus={autoFocus}
            autoComplete={autoComplete}
            status={error ? 'error' : undefined}
            aria-invalid={Boolean(error)}
            aria-describedby={error || hint ? `${name}-message` : undefined}
          />
        )}
      />
    </Field>
  );
}
