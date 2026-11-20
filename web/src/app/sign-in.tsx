import { LockOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useSession } from '../api/session-context';
import { Alert } from '../components/alert';
import { Form } from '../components/form';
import { List, ListItem } from '../components/list';
import { Logo } from '../components/logo';
import { Heading, Text } from '../components/text';
import { TextInput } from '../components/text-input';
import { tokenSchema } from '../forms/schemas';
import type { TokenValues } from '../forms/schemas';

// A token, proved against /v1/whoami before it is kept: a field that accepts a token the server refuses
// teaches the wrong thing about where the problem is.
//
// The screen is a split, not a card floating in the middle of an empty page: two thirds of paper that
// say what this is, one third of orange that holds the one thing you can do here. At 360px the split
// collapses and the form keeps the whole width, because the pitch is decoration and the form is not.
// The right side moves — an aura behind the card and a slow ring — because a still gradient reads as a
// screenshot, and this is the page where somebody decides whether to trust the thing.

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
          <Logo size={40} />
          noXis
        </div>

        <div className="max-w-xl">
          <Heading level={1} className="text-[32px] leading-tight">
            Data governance for the databases you already run
          </Heading>
          <Text as="p" size="lg" tone="muted" className="mt-4 leading-relaxed">
            An agent reads, and proposes. A rehearsal proves the change in a throwaway database, and
            a person holding <code>approve</code> decides. Nothing here happens because a model was
            confident.
          </Text>
          <List className="mt-6">
            <ListItem>Read and propose, never apply: the two scopes are two people.</ListItem>
            <ListItem>Every change is rehearsed, diffed, and rolled back before review.</ListItem>
            <ListItem>Every action is journaled with the name of the caller that took it.</ListItem>
          </List>
        </div>

        <Text size="xs" tone="muted">
          One binary · one state file · no telemetry
        </Text>
      </section>

      <aside className="noxis-signin-panel relative flex flex-col justify-center overflow-hidden bg-brand px-6 py-12 md:px-10">
        <span aria-hidden="true" className="noxis-glow" />
        <span aria-hidden="true" className="noxis-glow noxis-glow-2" />
        <svg aria-hidden="true" className="noxis-orbit" viewBox="0 0 600 600" fill="none">
          <defs>
            <linearGradient id="noxis-orbit-a" x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stopColor="#ffffff" stopOpacity="0" />
              <stop offset="50%" stopColor="#ffffff" stopOpacity="0.55" />
              <stop offset="100%" stopColor="#ffffff" stopOpacity="0" />
            </linearGradient>
            <linearGradient id="noxis-orbit-b" x1="1" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#fed7aa" stopOpacity="0" />
              <stop offset="50%" stopColor="#fed7aa" stopOpacity="0.45" />
              <stop offset="100%" stopColor="#fed7aa" stopOpacity="0" />
            </linearGradient>
          </defs>
          <circle className="noxis-orbit-track" cx="300" cy="300" r="252" />
          <circle className="noxis-orbit-track" cx="300" cy="300" r="198" />
          <circle
            className="noxis-orbit-arc"
            cx="300"
            cy="300"
            r="252"
            stroke="url(#noxis-orbit-a)"
            strokeDasharray="150 1433"
          />
          <circle
            className="noxis-orbit-arc noxis-orbit-arc-2"
            cx="300"
            cy="300"
            r="198"
            stroke="url(#noxis-orbit-b)"
            strokeDasharray="94 1150"
          />
        </svg>

        <div className="relative mx-auto w-full max-w-sm rounded-2xl bg-white p-6 shadow-xl shadow-orange-900/20 sm:p-7">
          <div className="mb-6 flex items-center gap-2.5 text-base font-bold tracking-tight md:hidden">
            <Logo size={32} tone="onBrand" />
            noXis
          </div>

          <Heading level={2}>Sign in</Heading>
          <Text as="p" size="sm" tone="muted" className="mt-1 mb-5">
            {config ? (
              <>
                to <code>{config.api.replace(/^https?:\/\//, '')}</code>
              </>
            ) : (
              'with the token your CLI holds'
            )}
          </Text>

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

          <Text as="p" size="xs" tone="muted" className="mt-4 leading-relaxed">
            The CLI stores the same token. Run <code>noxis web serve</code> and it is already here.
          </Text>
        </div>
      </aside>
    </div>
  );
}
