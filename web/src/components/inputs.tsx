import { InputNumber, Select } from 'antd';
import { Controller, useFormFields } from './form';
import { Field } from './field';

function messageOf(errors: Record<string, unknown>, name: string): string | undefined {
  const found = errors[name];
  if (found && typeof found === 'object' && 'message' in found) {
    return String((found as { message?: unknown }).message ?? '');
  }
  return undefined;
}

// A number that the schema can coerce, and a choice that only offers what the API accepts: the options
// are the caller's list, so a value the server would refuse is never on screen.

export function NumberInput({
  name,
  label,
  hint,
  min,
  max,
  fullWidth = true,
}: {
  name: string;
  label: string;
  hint?: string;
  min?: number;
  max?: number;
  fullWidth?: boolean;
}) {
  const { control, errors } = useFormFields();
  const error = messageOf(errors as Record<string, unknown>, name);
  return (
    <Field label={label} htmlFor={name} error={error} hint={hint} fullWidth={fullWidth}>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <InputNumber
            id={name}
            min={min}
            max={max}
            className="w-full"
            value={typeof field.value === 'number' ? field.value : undefined}
            onChange={(next) => field.onChange(next ?? 0)}
            status={error ? 'error' : undefined}
            aria-invalid={Boolean(error)}
            aria-describedby={error || hint ? `${name}-message` : undefined}
          />
        )}
      />
    </Field>
  );
}

export function SelectInput({
  name,
  label,
  hint,
  options,
  fullWidth = true,
}: {
  name: string;
  label: string;
  hint?: string;
  options: { value: string; label: string }[];
  fullWidth?: boolean;
}) {
  const { control, errors } = useFormFields();
  const error = messageOf(errors as Record<string, unknown>, name);
  return (
    <Field label={label} htmlFor={name} error={error} hint={hint} fullWidth={fullWidth}>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Select
            id={name}
            options={options}
            className="w-full"
            value={field.value as string}
            onChange={(next) => field.onChange(next)}
            status={error ? 'error' : undefined}
            aria-invalid={Boolean(error)}
          />
        )}
      />
    </Field>
  );
}
