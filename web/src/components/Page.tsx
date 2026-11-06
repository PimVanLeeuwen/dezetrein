import { useEffect, useRef, type ReactNode } from 'react';
import { useI18n } from '../i18n/i18n';

// Page sets the document title and, after an in-app navigation, moves focus to the heading so
// screen readers announce the new page (as a full page load would).
export function Page({ title, children, lead }: { title: string; lead?: ReactNode; children?: ReactNode }) {
  const { t } = useI18n();
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    document.title = title === t('app.name') ? title : `${title} · ${t('app.name')}`;
  }, [title, t]);
  useEffect(() => {
    if (document.activeElement && document.activeElement !== document.body) heading.current?.focus();
  }, [title]);
  return (
    <article className="pt-6">
      <h1 ref={heading} tabIndex={-1} className="text-3xl font-bold tracking-tight outline-none sm:text-4xl">
        {title}
      </h1>
      {lead && <div className="mt-3 text-lg text-neutral-700 dark:text-neutral-300">{lead}</div>}
      {children}
    </article>
  );
}
