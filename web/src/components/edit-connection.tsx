import { App as AntApp, Drawer } from 'antd';
import { useState } from 'react';
import { useProbeTarget, useTestConnection, useUpdateTarget } from '../api/queries';
import { Alert } from './alert';
import { AppButton } from './app-button';
import { Form, useFormFields } from './form';
import { SelectInput } from './inputs';
import { RenderWhen } from './render-when';
import { Text } from './text';
import { TextInput } from './text-input';
import { connectionDefaults, connectionSchema, tlsOptions } from '../forms/schemas';
import type { ConnectionValues } from '../forms/schemas';
import type { Target } from '../api/types';

// A connection that was registered can be corrected. What it changes is what the row says — the name,
// the credentials, the TLS mode — and not where it points: host, port and schema are the instance's
// identity, and a moved server is a new instance with its own proposals.

const FORM_ID = 'edit-connection';

export function EditConnection({
  target,
  open,
  onClose,
}: {
  target: Target;
  /** The drawer is controlled from the header's menu: one trigger for every action on the instance. */
  open: boolean;
  onClose: () => void;
}) {
  return (
    <Drawer
      title="Edit connection"
      open={open}
      onClose={onClose}
      width={Math.min(460, window.innerWidth)}
      destroyOnHidden
      footer={
        <div className="flex justify-end gap-2">
          <AppButton onClick={onClose}>Cancel</AppButton>
          <AppButton intent="primary" htmlType="submit" form={FORM_ID}>
            Save
          </AppButton>
        </div>
      }
    >
      <ConnectionForm target={target} onSaved={onClose} />
    </Drawer>
  );
}

function ConnectionForm({ target, onSaved }: { target: Target; onSaved: () => void }) {
  const update = useUpdateTarget(target.ref);
  const probe = useProbeTarget();
  const { message } = AntApp.useApp();

  return (
    <Form<ConnectionValues>
      id={FORM_ID}
      schema={connectionSchema}
      defaultValues={connectionDefaults(target)}
      onSubmit={async (values) => {
        const body: Record<string, unknown> = {
          name: values.name,
          username: values.username,
          tls_mode: values.tls_mode,
        };
        // Empty is "leave it": the server never reads the stored password back, so a blank box cannot
        // mean "set the password to nothing".
        if (values.password) body.password = values.password;
        await update.mutateAsync(body);
        // The probe after the save is what tells the operator the credential works in the instance's
        // own context. It is best effort: the save already happened, and a failed probe is a fact
        // about the connection rather than a failed save.
        probe.mutate(target.ref);
        void message.success('saved');
        onSaved();
      }}
    >
      <TextInput name="name" label="Name" autoFocus />
      <TextInput name="username" label="Username" autoComplete="off" />
      <TextInput
        name="password"
        label="Password"
        password
        hint="Leave it empty to keep the stored one."
        autoComplete="new-password"
      />
      <SelectInput name="tls_mode" label="TLS" options={tlsOptions} />
      <ProbeCredentials target={target} />
      <Text size="xs" tone="muted">
        {target.host}:{target.port} · {target.database} — a moved server is a new instance, so host,
        port and schema are not edited here.
      </Text>
    </Form>
  );
}

// The CLI's rule, in the console: the new credentials are dialled before they are saved, and the stored
// password is never read back, so a test needs the operator to type one.
function ProbeCredentials({ target }: { target: Target }) {
  const { getValues, watch } = useFormFields();
  const test = useTestConnection();
  const [answer, setAnswer] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const typed = String(watch('password') ?? '') !== '';

  const probe = async () => {
    setAnswer(null);
    setFailure(null);
    const values = getValues() as unknown as ConnectionValues;
    try {
      const result = await test.mutateAsync({
        driver: target.driver,
        host: target.host,
        port: target.port,
        database: target.database,
        username: values.username,
        password: values.password,
        tls_mode: values.tls_mode,
        options: target.options ?? {},
      });
      setAnswer(`Answered as ${result.capabilities.version || result.capabilities.driver}`);
    } catch (error) {
      setFailure(error instanceof Error ? error.message : String(error));
    }
  };

  return (
    <div className="mb-4 flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        <AppButton
          htmlType="button"
          onClick={() => void probe()}
          loading={test.isPending}
          disabled={!typed}
        >
          Test connection
        </AppButton>
        <Text size="xs" tone="muted">
          {typed
            ? 'Dials the values above and writes nothing'
            : 'Type the password to test it: the stored one is never read back'}
        </Text>
      </div>
      <RenderWhen condition={answer}>
        <Text size="xs" tone="muted">
          {answer}
        </Text>
      </RenderWhen>
      <RenderWhen condition={failure}>
        <Alert tone="error" title="The credentials did not answer" description={failure} />
      </RenderWhen>
    </div>
  );
}
