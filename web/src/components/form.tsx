import { createContext, useContext } from 'react';
import { Controller, useForm } from 'react-hook-form';
import type { Control, DefaultValues, FieldErrors, FieldValues, Resolver } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import type { ReactNode } from 'react';
import type { z } from 'zod';
import { AppButton } from './app-button';

// The form every screen uses: a schema, a submit handler, and fields that find their own value and
// their own error. A page never calls useForm, never names a control, and never renders an error.

type Fields = {
  control: Control<FieldValues>;
  errors: FieldErrors<FieldValues>;
  submitting: boolean;
};

const FormContext = createContext<Fields | null>(null);

export function useFormFields(): Fields {
  const fields = useContext(FormContext);
  if (!fields) throw new Error('a field must live inside <Form>');
  return fields;
}

export function Form<T extends FieldValues>({
  schema,
  defaultValues,
  onSubmit,
  children,
  className,
  id,
}: {
  schema: z.ZodType<T>;
  defaultValues: DefaultValues<T>;
  onSubmit: (values: T) => void | Promise<void>;
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  const methods = useForm<T>({
    resolver: zodResolver(schema) as unknown as Resolver<T>,
    mode: 'onChange',
    defaultValues,
  });

  return (
    <FormContext.Provider
      value={{
        control: methods.control as unknown as Control<FieldValues>,
        errors: methods.formState.errors as unknown as FieldErrors<FieldValues>,
        submitting: methods.formState.isSubmitting,
      }}
    >
      <form
        id={id}
        noValidate
        className={className}
        onSubmit={methods.handleSubmit((values) => onSubmit(values as T))}
      >
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
