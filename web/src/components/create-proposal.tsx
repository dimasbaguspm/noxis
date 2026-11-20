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
// The engine decides what the statement means — a data change runs in a transaction, a schema change is
// a single statement — and the form does not ask which, because the SQL already says. The rollback is
// the point of the form, not an extra field, so it sits directly under the query with a sentence about
// what an empty one means.

export function CreateProposal({ target, schema }: { target: string; schema: Schema | null }) {
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const create = useCreateProposal(target);

  return (
    <>
      <Can do="propose">
        <AppButton intent="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>
          New change
        </AppButton>
      </Can>

      <Modal
        open={open}
        title="A data change"
        onCancel={() => setOpen(false)}
        footer={null}
        width="min(920px, 94vw)"
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
            // A filed change whose rehearsal failed is not a success: the CLI exits non-zero on the same
            // answer, so the console says it too instead of a green tick over a failed proposal.
            if (filed.status === 'failed') {
              void message.warning({
                content: `${filed.title} is filed, but the rehearsal failed`,
                duration: 6,
              });
            } else {
              void message.success(`${filed.title} filed`);
            }
            setOpen(false);
          }}
        >
          <TextInput
            name="title"
            label="Title"
            autoFocus
            placeholder="add a column for the audit flag"
          />
          <SqlField name="query" label="Query" schema={schema} height={220} />
          <SqlField
            name="rollback"
            label="Rollback"
            height={140}
            hint="Leave it empty only if this cannot be undone: an apply then has to acknowledge that by name."
          />
          <Form.Submit block>File the proposal</Form.Submit>
        </Form>
      </Modal>
    </>
  );
}
