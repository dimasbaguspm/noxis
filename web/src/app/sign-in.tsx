import { LockOutlined } from '@ant-design/icons';
import { Alert, Card } from 'antd';
import { useState } from 'react';
import { useSession } from '../api/session-context';
import { AppButton } from '../components/app-button';
import { Form } from '../components/form';
import { TextInput } from '../components/text-input';
import { tokenSchema } from '../forms/schemas';
import type { TokenValues } from '../forms/schemas';

// A token, proved against /v1/whoami before it is kept: a card that accepts a token the server refuses
// teaches the wrong thing about where the problem is.
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
    <div className="grid min-h-[70vh] place-items-center px-4">
      <Card className="w-full max-w-sm">
        <h1 className="m-0 text-lg font-semibold">Sign in</h1>
        <p className="mt-1 mb-5 text-sm text-muted">
          {config ? (
            <>
              to <span className="font-mono">{config.api.replace(/^https?:\/\//, '')}</span>
            </>
          ) : (
            'to noXis'
          )}
        </p>

        {refused ? (
          <Alert type="error" showIcon message={refused} className="mb-4" role="alert" />
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

        <p className="mt-4 mb-0 text-xs text-muted">
          The CLI stores the same token. Run <span className="font-mono">noxis web serve</span> and it
          is already here.
        </p>
      </Card>
    </div>
  );
}

export { AppButton };
