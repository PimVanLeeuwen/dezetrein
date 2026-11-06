import type { PageContent } from '../content/pages';
import { useI18n } from '../i18n/i18n';
import { focusRing } from './Layout';
import { Page } from './Page';

// ContentPage renders a privacy or about text: a heading per section, on the rail line's paper.
// Links leave the site, so they open in a new tab and carry no referrer.
export function ContentPage({ content }: { content: PageContent }) {
  const { t } = useI18n();
  return (
    <Page title={content.title} lead={<p>{content.lead}</p>}>
      <div className="mt-10 flex max-w-xl flex-col gap-9">
        {content.sections.map((s, i) => (
          <section key={s.heading} className="rise" style={{ animationDelay: `${100 + i * 40}ms` }}>
            <h2 className="font-display text-2xl font-bold tracking-[-0.02em]">{s.heading}</h2>
            {s.paragraphs.map((parts, j) => (
              <p key={j} className="mt-2 text-lg leading-relaxed">
                {parts.map((part, k) =>
                  typeof part === 'string' ? (
                    part
                  ) : (
                    <a key={k} href={part.href} target="_blank" rel="noopener noreferrer" className={`rounded font-bold text-line underline underline-offset-4 ${focusRing}`}>
                      {part.text}
                      <span aria-hidden="true"> ↗</span>
                      <span className="sr-only"> {t('content.newTab')}</span>
                    </a>
                  ),
                )}
              </p>
            ))}
          </section>
        ))}
        <p className="text-sm font-medium text-dim">{content.updated}</p>
      </div>
    </Page>
  );
}
