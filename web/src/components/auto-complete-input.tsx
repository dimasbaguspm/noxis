import { AutoComplete } from 'antd';
import { Controller, useFormFields } from './form';
import { Field } from './field';

// A name the server answered, that may also be typed. Ant Design's Select refuses a value it was not
// offered; a connection that could not list its schemas is still a connection somebody wants to
// register, so this control offers and accepts.
export function AutoCompleteInput({
  name,
  label,
  hint,
  options,
  placeholder,
}: {
  name: string;
  label: string;
  hint?: string;
  /** What the server said it holds; empty means the field is just a text box. */
  options: string[];
  placeholder?: string;
}) {
  const { control, errors } = useFormFields();
  const error = messageOf(errors as Record<string, unknown>, name);

  return (
    <Field label={label} htmlFor={name} error={error} hint={hint} fullWidth>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <AutoComplete
            id={name}
            className="w-full"
            value={(field.value as string) ?? ''}
            options={options.map((option) => ({ value: option, label: option }))}
            onChange={(next) => field.onChange(next ?? '')}
            placeholder={placeholder}
            filterOption={(input, option) =>
              (option?.value ?? '').toLowerCase().includes(input.toLowerCase())
            }
            status={error ? 'error' : undefined}
            aria-invalid={Boolean(error)}
            aria-describedby={error || hint ? `${name}-message` : undefined}
          />
        )}
      />
    </Field>
  );
}

function messageOf(errors: Record<string, unknown>, name: string): string | undefined {
  const found = errors[name];
  if (found && typeof found === 'object' && 'message' in found) {
    return String((found as { message?: unknown }).message ?? '');
  }
  return undefined;
}
