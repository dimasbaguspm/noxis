import { clsx } from 'clsx';
import { Controller, useFormFields } from './form';
import { SqlEditor } from '../editor/sql-editor';
import type { Schema } from '../editor/sql-editor';

// A SQL field inside a form: the same Monaco the Query tab uses, wired to react-hook-form through a
// Controller, because a bare `register` never sees the editor's value. The label and the error are
// the same as every other field's, so a proposal form looks like the rest of the console.

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
    <div className="mb-4">
      <label htmlFor={name} className="mb-1.5 block text-sm font-medium text-ink">
        {label}
      </label>
      <Controller
        name={name}
        control={control}
        render={({ field }) => (
          <div
            id={name}
            className={clsx(
              'overflow-hidden rounded-lg border',
              error ? 'border-red-400' : 'border-line',
            )}
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
      {error ? (
        <p role="alert" className="mt-1 text-xs text-red-600">
          {error}
        </p>
      ) : hint ? (
        <p className="mt-1 text-xs text-muted">{hint}</p>
      ) : null}
    </div>
  );
}
