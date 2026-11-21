import { Select } from 'antd';

// One name out of a list the server answered. Ant Design's Select is the right control and its props
// are not: the console only ever picks one name, so that is the whole API here.
export function Picker({
  label,
  value,
  options,
  onChange,
  placeholder,
  loading = false,
}: {
  /** The control's accessible name, since the visible label is often the card's title. */
  label: string;
  value: string | null;
  options: string[];
  onChange: (next: string) => void;
  placeholder?: string;
  loading?: boolean;
}) {
  return (
    <Select
      aria-label={label}
      className="w-full"
      value={value ?? undefined}
      onChange={onChange}
      loading={loading}
      showSearch
      optionFilterProp="label"
      placeholder={placeholder ?? 'nothing to pick'}
      options={options.map((option) => ({ value: option, label: option }))}
    />
  );
}
