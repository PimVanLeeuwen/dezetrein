import { useEffect, useRef, type ReactNode } from 'react';
import { useI18n } from '../i18n/i18n';

// Page sets the document title and, after an in-app navigation, moves focus to the heading so
// screen readers announce the new page (as a full page load would). heading replaces the plain
// title text when the page wants an accent in it; compact is for long headings, such as questions.
export function Page({ title, heading, lead, compact, children }: { title: string; heading?: ReactNode; lead?: ReactNode; compact?: boolean; children?: ReactNode }) {
  const { t } = useI18n();
  const ref = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    document.title = title === t('app.name') ? title : `${title} · ${t('app.name')}`;
  }, [title, t]);
  useEffect(() => {
    if (document.activeElement && document.activeElement !== document.body) ref.current?.focus();
  }, [title]);
  return (
    <article className="pt-8 sm:pt-14">
      <h1
        ref={ref}
        tabIndex={-1}
        className={`rise font-display leading-[0.95] font-extrabold tracking-[-0.04em] text-balance outline-none ${compact ? 'text-[2.6rem] sm:text-6xl' : 'text-[3.25rem] leading-[0.92] tracking-[-0.045em] sm:text-7xl'}`}
      >
        {heading ?? title}
      </h1>
      {lead && (
        <div className="rise mt-5 max-w-md text-lg leading-snug font-medium text-dim sm:text-xl" style={{ animationDelay: '80ms' }}>
          {lead}
        </div>
      )}
      {children}
    </article>
  );
}
