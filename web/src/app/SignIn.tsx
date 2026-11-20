import { Alert, Button, Card, Form, Input, Typography } from 'antd';
import { LockOutlined } from '@ant-design/icons';
import { useState } from 'react';
import { useSession } from '../api/session';

// A token, once. The console proves it against /v1/whoami before it keeps it: a card that accepts a
// token the server refuses is a card that teaches the wrong thing about where the problem is.

export function SignIn() {
  const { signIn, config, error } = useSession();
  const [busy, setBusy] = useState(false);
  const [refused, setRefused] = useState<string | null>(null);

  const submit = async ({ token }: { token: string }) => {
    setBusy(true);
    setRefused(null);
    try {
      await signIn(token);
    } catch (cause) {
      setRefused(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="noxis-signin">
      <Card className="noxis-signin-card">
        <Typography.Title level={3} style={{ marginTop: 0 }}>
          Sign in to noXis
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 20 }}>
          Paste a token the API accepts. The CLI already has one:{' '}
          <Typography.Text code>noxis auth signin &lt;token&gt;</Typography.Text> stores it, and the
          token is the same string.
        </Typography.Paragraph>

        {config ? (
          <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
            This console calls <Typography.Text code>{config.api}</Typography.Text>
          </Typography.Paragraph>
        ) : null}

        {error && !refused ? <Alert type="warning" showIcon message={error} style={{ marginBottom: 16 }} /> : null}
        {refused ? <Alert type="error" showIcon message={refused} style={{ marginBottom: 16 }} /> : null}

        <Form layout="vertical" onFinish={submit} requiredMark={false}>
          <Form.Item
            name="token"
            label="Token"
            rules={[{ required: true, message: 'a token is required' }]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder="the token the API was given" autoFocus />
          </Form.Item>
          <Button type="primary" htmlType="submit" block loading={busy}>
            Sign in
          </Button>
        </Form>
      </Card>
    </div>
  );
}
