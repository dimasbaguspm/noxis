import { PlusOutlined } from '@ant-design/icons';
import { App as AntApp, Modal } from 'antd';
import { useState } from 'react';
import { Can } from '../api/permissions';
import { useCreateProposal } from '../api/queries';
import { AppButton } from './app-button';
import { Form } from './form';
import { SqlField } from './sql-field';
import { TextInput } from './text-input';
import { proposalDefaults, proposalSchema } from '../forms/schemas';
import type { ProposalValues } from '../forms/schemas';
import type { Schema } from '../editor/sql-editor';

// Filing a change from the instance page, which is where an operator already is when they think of one.
// The kind is the tab it was filed from, and the API decides what that means: a DML proposal runs in a
// transaction, a DDL one is a single statement. The rollback is the point of the form, not an extra
// field, so it sits directly under the statement with a sentence about what an empty one means.

export function CreateProposal({
  target,
  kind,
  schema,
}: {
  target: string;
  kind: 'dml' | 'ddl';
  schema: Schema | null;
}) {
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const create = useCreateProposal(target);

  return (
    <>
      <Can do="propose">
        <AppButton intent="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          New {kind === 'ddl' ? 'DDL' : 'DML'}
        </AppButton>
      </Can>

      <Modal
        open={open}
        title={`A ${kind === 'ddl' ? 'schema' : 'data'} change`}
        onCancel={() => setOpen(false)}
        footer={null}
        width={720}
        destroyOnHidden
      >
        <Form<ProposalValues>
          schema={proposalSchema}
          defaultValues={proposalDefaults}
          onSubmit={async (values) => {
            const filed = await create.mutateAsync({
              title: values.title,
              query: values.query,
              rollback: values.rollback || undefined,
            });
            void message.success(`${filed.title} filed`);
            setOpen(false);
          }}
        >
          <TextInput
            name="title"
            label="Title"
            autoFocus
            placeholder="add a column for the audit flag"
          />
          <SqlField name="query" label="Query" schema={schema} height={170} />
          <SqlField
            name="rollback"
            label="Rollback"
            height={110}
            hint="Leave it empty only if this cannot be undone: an apply then has to acknowledge that by name."
          />
          <Form.Submit block>File the proposal</Form.Submit>
        </Form>
      </Modal>
    </>
  );
}
