import { LockOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useSession } from '../api/session-context';
import { Alert } from '../components/alert';
import { Form } from '../components/form';
import { TextInput } from '../components/text-input';
import { tokenSchema } from '../forms/schemas';
import type { TokenValues } from '../forms/schemas';

// A token, proved against /v1/whoami before it is kept: a field that accepts a token the server refuses
// teaches the wrong thing about where the problem is.
//
// The screen is a split, not a card floating in the middle of an empty page: two thirds of paper that
// say what this is, one third of orange that holds the one thing you can do here. At 360px the split
// collapses and the form keeps the whole width, because the pitch is decoration and the form is not.

export function SignIn() {
  const { signIn, config } = useSession();
  const [refused, setRefused] = useState<string | null>(null);

  const submit = async (values: TokenValues) => {
    setRefused(null);
    try {
      await signIn(values.token);
    } catch (cause) {
      setRefused(cause instanceof Error ? cause.message : String(cause));
    }
  };

  return (
    <div className="grid min-h-dvh grid-cols-1 md:grid-cols-[65fr_35fr]">
      <section className="hidden flex-col justify-between gap-10 bg-white p-10 md:flex lg:p-14">
        <div className="flex items-center gap-2.5 text-base font-bold tracking-tight">
          <span aria-hidden="true" className="h-2.5 w-2.5 rounded-sm bg-brand" />
          noXis
        </div>

        <div className="max-w-xl">
          <h1 className="m-0 text-[32px] leading-tight font-semibold tracking-tight">
            Reviewed access to MySQL and MariaDB
          </h1>
          <p className="mt-4 mb-0 text-base leading-relaxed text-muted">
            An agent reads, and proposes. A rehearsal proves the change in a throwaway database, and
            a person holding <span className="font-mono text-[13px] text-ink">approve</span>{' '}
            decides. Nothing here happens because a model was confident.
          </p>
        </div>

        <p className="m-0 text-xs text-muted">MySQL 8 · MariaDB 11 · one binary, one state file</p>
      </section>

      <aside className="flex flex-col justify-center bg-brand px-6 py-12 md:px-10">
        <div className="mx-auto w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl shadow-orange-900/10 sm:p-7">
          <div className="mb-6 flex items-center gap-2.5 text-base font-bold tracking-tight md:hidden">
            <span aria-hidden="true" className="h-2.5 w-2.5 rounded-sm bg-brand" />
            noXis
          </div>

          <h2 className="m-0 text-lg font-semibold">Sign in</h2>
          <p className="mt-1 mb-5 text-sm text-muted">
            {config ? (
              <>
                to{' '}
                <span className="font-mono text-[13px]">
                  {config.api.replace(/^https?:\/\//, '')}
                </span>
              </>
            ) : (
              'to noXis'
            )}
          </p>

          {refused ? (
            <Alert
              tone="error"
              title="That token was refused"
              description={refused}
              className="mb-4"
            />
          ) : null}

          <Form schema={tokenSchema} defaultValues={{ token: '' }} onSubmit={submit}>
            <TextInput
              name="token"
              label="Token"
              password
              autoFocus
              autoComplete="off"
              icon={<LockOutlined />}
            />
            <Form.Submit block>Continue</Form.Submit>
          </Form>

          <p className="mt-4 mb-0 text-xs leading-relaxed text-muted">
            The CLI stores the same token. Run <span className="font-mono">noxis web serve</span>{' '}
            and it is already here.
          </p>
        </div>
      </aside>
    </div>
  );
}
