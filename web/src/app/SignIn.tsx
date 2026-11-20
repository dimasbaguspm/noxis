import { Button, Card, Form, Typography } from 'antd';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useState } from 'react';
import { useSession } from '../api/session';
import { tokenSchema } from '../forms/schemas';
import type { TokenValues } from '../forms/schemas';
import { TextField } from '../components/Field';

// A token, once. Proved against /v1/whoami before it is kept: a card that accepts a token the server
// refuses teaches the wrong thing about where the problem is.

export function SignIn() {
  const { signIn, config } = useSession();
  const [refused, setRefused] = useState<string | null>(null);
  const form = useForm<TokenValues>({
    resolver: zodResolver(tokenSchema),
    mode: 'onChange',
    defaultValues: { token: '' },
  });

  const submit = form.handleSubmit(async ({ token }) => {
    setRefused(null);
    try {
      await signIn(token);
    } catch (cause) {
      setRefused(cause instanceof Error ? cause.message : String(cause));
    }
  });

  return (
    <div className="noxis-signin">
      <Card className="noxis-signin-card">
        <Typography.Title level={4} style={{ marginTop: 0, marginBottom: 4 }}>
          Sign in
        </Typography.Title>
        <Typography.Text type="secondary">
          {config ? (
            <>
              to <Typography.Text code>{config.api.replace(/^https?:\/\//, '')}</Typography.Text>
            </>
          ) : (
            'to noXis'
          )}
        </Typography.Text>

        <Form layout="vertical" onFinish={submit} style={{ marginTop: 20 }}>
          <TextField
            control={form.control}
            name="token"
            label="Token"
            password
            autoFocus
            autoComplete="off"
            error={form.formState.errors.token?.message}
            hint={refused ?? undefined}
          />
          <Button type="primary" htmlType="submit" block loading={form.formState.isSubmitting}>
            Continue
          </Button>
        </Form>

        <Typography.Paragraph type="secondary" style={{ marginTop: 16, marginBottom: 0, fontSize: 12 }}>
          <Typography.Text code>noxis auth signin</Typography.Text> stores the same token for the CLI;
          run <Typography.Text code>noxis web serve</Typography.Text> and it is already here.
        </Typography.Paragraph>
      </Card>
    </div>
  );
}
