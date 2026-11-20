import { Input as AntInput, Typography } from 'antd';
import { Controller } from 'react-hook-form';
import type { Control, FieldValues, Path } from 'react-hook-form';
import type { ReactNode } from 'react';

// One field: the label, the control, and the one line of message under it. Ant Design's Form gives
// this for free only when the form owns the state; react-hook-form owns it here, so the layout is a
// component instead of a convention every form has to remember.
export function Field({
  label,
  error,
  hint,
  children,
}: {
  label: string;
  error?: string;
  hint?: string;
  children: ReactNode;
}) {
  const message = error ?? hint;
  return (
    <div style={{ marginBottom: 16 }}>
      <Typography.Text style={{ display: 'block', marginBottom: 6, fontSize: 13, fontWeight: 500 }}>
        {label}
      </Typography.Text>
      {children}
      {message ? (
        <Typography.Text
          type={error ? 'danger' : 'secondary'}
          style={{ display: 'block', marginTop: 6, fontSize: 12 }}
        >
          {message}
        </Typography.Text>
      ) : null}
    </div>
  );
}

// antd's Input is a controlled component and react-hook-form's `register` is not enough for it: the
// value never reaches the resolver that way, and a form that says "required" with text on screen is
// worse than no validation. Controller is the bridge, so every text input goes through this one.
export function TextField<T extends FieldValues>({
  control,
  name,
  label,
  error,
  hint,
  password,
  placeholder,
  autoFocus,
  autoComplete,
}: {
  control: Control<T>;
  name: Path<T>;
  label: string;
  error?: string;
  hint?: string;
  password?: boolean;
  placeholder?: string;
  autoFocus?: boolean;
  autoComplete?: string;
}) {
  const Input = password ? AntInput.Password : AntInput;
  return (
    <Field label={label} error={error} hint={hint}>
      <Controller
        control={control}
        name={name}
        render={({ field }) => (
          <Input
            {...field}
            value={(field.value as string) ?? ''}
            onChange={(event) => field.onChange(event.target.value)}
            placeholder={placeholder}
            autoFocus={autoFocus}
            autoComplete={autoComplete}
          />
        )}
      />
    </Field>
  );
}
