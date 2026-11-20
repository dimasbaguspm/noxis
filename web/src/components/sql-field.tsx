import { Controller, useFormFields } from './form';
import { Field } from './field';
import { SqlEditor } from '../editor/sql-editor';
import type { Schema } from '../editor/sql-editor';

// A SQL field inside a form: the same Monaco the Query tab uses, wired to react-hook-form through a
// Controller, because a bare `register` never sees the editor's value. It renders the same `Field` a
// text input does, so the label, the gap and the error line are identical wherever a form mixes them.

function messageOf(errors: Record<string, unknown>, name: string): string | undefined {
  const found = errors[name];
  if (found && typeof found === 'object' && 'message' in found) {
    return String((found as { message?: unknown }).message ?? '');
  }
  return undefined;
}

export function SqlField({
  name,
  label,
  height = 140,
  schema = null,
  hint,
}: {
  name: string;
  label: string;
  height?: number;
  schema?: Schema | null;
  hint?: string;
}) {
  const { control, errors } = useFormFields();
  const error = messageOf(errors, name);

  return (
    <Field label={label} htmlFor={name} error={error} hint={hint} fullWidth>
      <Controller
        name={name}
        control={control}
        render={({ field }) => (
          <div
            id={name}
            className={`overflow-hidden rounded-lg border ${error ? 'border-red-400' : 'border-line'}`}
          >
            <SqlEditor
              value={String(field.value ?? '')}
              onChange={(value) => field.onChange(value ?? '')}
              schema={schema}
              height={height}
            />
          </div>
        )}
      />
    </Field>
  );
}
