import { createContext, useContext, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import type { Control, DefaultValues, FieldErrors, FieldValues, Resolver } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import type { ReactNode } from 'react';
import type { z } from 'zod';
import { Alert } from './alert';
import { AppButton } from './app-button';

// The form every screen uses: a schema, a submit handler, and fields that find their own value and
// their own error. A page never calls useForm, never names a control, and never renders an error.
//
// Two kinds of failure land in the banner above the first field, because the field-level message only
// says which box is wrong: a rejected submit (the server said no) and a schema that was not satisfied.

type Fields = {
  control: Control<FieldValues>;
  errors: FieldErrors<FieldValues>;
  submitting: boolean;
  /** Every value as it stands: a button inside the form that dials with them reads this. */
  getValues: () => FieldValues;
  /** The one it may fill in, when the server answered a name the operator would have typed. */
  setValue: (name: string, value: string) => void;
};

const FormContext = createContext<Fields | null>(null);

export function useFormFields(): Fields {
  const fields = useContext(FormContext);
  if (!fields) throw new Error('a field must live inside <Form>');
  return fields;
}

function sentenceOf(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === 'string') return error;
  return 'That did not work';
}

/** The first message react-hook-form produced, wherever in the tree it sits. */
function firstMessage(errors: FieldErrors<FieldValues>): string | null {
  for (const value of Object.values(errors)) {
    if (!value) continue;
    const nested = value as { message?: unknown };
    if (typeof nested.message === 'string') return nested.message;
    const deeper = firstMessage(value as FieldErrors<FieldValues>);
    if (deeper) return deeper;
  }
  return null;
}

export function Form<T extends FieldValues>({
  schema,
  defaultValues,
  onSubmit,
  onSubmitError,
  children,
  className,
  id,
}: {
  schema: z.ZodType<T>;
  defaultValues: DefaultValues<T>;
  onSubmit: (values: T) => void | Promise<void>;
  /** Called after the banner is shown, with the values that were submitted and what was thrown. */
  onSubmitError?: (values: T, error: unknown) => void;
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  const methods = useForm<T>({
    resolver: zodResolver(schema) as unknown as Resolver<T>,
    mode: 'onChange',
    defaultValues,
  });
  const [failure, setFailure] = useState<{ title: string; detail?: string } | null>(null);

  return (
    <FormContext.Provider
      value={{
        control: methods.control as unknown as Control<FieldValues>,
        errors: methods.formState.errors as unknown as FieldErrors<FieldValues>,
        submitting: methods.formState.isSubmitting,
        getValues: () => methods.getValues() as FieldValues,
        setValue: methods.setValue as unknown as (name: string, value: string) => void,
      }}
    >
      <form
        id={id}
        noValidate
        className={className}
        onSubmit={methods.handleSubmit(
          async (values) => {
            setFailure(null);
            try {
              await onSubmit(values as T);
            } catch (error) {
              setFailure({ title: sentenceOf(error) });
              onSubmitError?.(values as T, error);
            }
          },
          (errors) => {
            setFailure({
              title: 'Some fields need attention',
              detail: firstMessage(errors as FieldErrors<FieldValues>) ?? undefined,
            });
          },
        )}
      >
        {failure ? (
          <Alert tone="error" title={failure.title} description={failure.detail} className="mb-4" />
        ) : null}
        {children}
      </form>
    </FormContext.Provider>
  );
}

// Form.Submit renders inside the form but still knows whether it is submitting: the alternative is
// every page reading formState to pass a loading flag into a button it already has.
Form.Submit = function Submit({ children, block }: { children: ReactNode; block?: boolean }) {
  const { submitting } = useFormFields();
  return (
    <AppButton htmlType="submit" intent="primary" block={block} loading={submitting}>
      {children}
    </AppButton>
  );
};

// Controller is re-exported so a field component can reach it without importing react-hook-form and
// the form at once; it is not part of the page-facing API.
export { Controller };
