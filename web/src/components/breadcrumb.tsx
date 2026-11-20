import { Link } from 'react-router-dom';

// A breadcrumb is where you are and how to leave: the last item is the page, the rest are links. A
// "back" button guesses; a crumb states.
export function Breadcrumb({ items }: { items: { label: string; to?: string }[] }) {
  return (
    <nav aria-label="Breadcrumb">
      <ol className="m-0 flex list-none flex-wrap items-center gap-1.5 p-0 text-sm">
        {items.map((item, index) => {
          const last = index === items.length - 1;
          return (
            <li key={`${item.label}-${index}`} className="flex items-center gap-1.5">
              {item.to && !last ? (
                <Link to={item.to} className="text-brand hover:underline">
                  {item.label}
                </Link>
              ) : (
                <span
                  aria-current={last ? 'page' : undefined}
                  className={last ? 'font-medium' : 'text-muted'}
                >
                  {item.label}
                </span>
              )}
              {last ? null : (
                <span aria-hidden="true" className="text-muted">
                  /
                </span>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
