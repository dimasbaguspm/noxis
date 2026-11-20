import { Button } from 'antd';
import { cva } from 'class-variance-authority';
import { clsx } from 'clsx';
import type { ButtonProps } from 'antd';

// One button for the console. The variant names the intent ("primary", "danger", "quiet") rather than
// Ant Design's `type`, and cva holds the class part so a new variant is a line rather than a branch.

const button = cva('', {
  variants: {
    intent: {
      primary: '',
      secondary: '',
      danger: '',
      quiet: '!border-transparent !bg-transparent !shadow-none',
    },
    block: {
      true: 'w-full',
      false: '',
    },
  },
  defaultVariants: { intent: 'secondary', block: false },
});

const types: Record<string, ButtonProps['type']> = {
  primary: 'primary',
  secondary: 'default',
  danger: 'primary',
  quiet: 'text',
};

export type AppButtonProps = Omit<ButtonProps, 'type' | 'danger'> & {
  intent?: 'primary' | 'secondary' | 'danger' | 'quiet';
  block?: boolean;
};

export function AppButton({ intent = 'secondary', block, className, ...rest }: AppButtonProps) {
  return (
    <Button
      type={types[intent]}
      danger={intent === 'danger'}
      className={clsx(button({ intent, block }), className)}
      {...rest}
    />
  );
}
