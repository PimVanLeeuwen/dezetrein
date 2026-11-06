import { useCallback, useEffect, useId, useRef, useState, type FormEvent } from 'react';
import { api as defaultClient, errorKey, type Client } from '../api/client';
import { Page } from '../components/Page';
import { numberInput, onlyDigits, primary, secondary, textLink } from '../components/ui';
import { useI18n } from '../i18n/i18n';
import type { MessageKey } from '../i18n/messages';
import { browserGeo, sample, type Geo } from '../lib/geo';
import { Link, navigate, paths } from '../lib/router';

// Searching by carriage (trainset) number. NS has no lookup by carriage, so the backend checks
// the trains it has seen recently first; when that finds nothing, a few location readings let it
// check the trains nearby. Found: straight to the trip, with the carriage marked.

export type CarriageState =
  | { phase: 'form' }
  | { phase: 'searching'; carriage: string }
  | { phase: 'needLocation'; carriage: string }
  | { phase: 'locating'; carriage: string }
  | { phase: 'notFound'; carriage: string; searched: number }
  | { phase: 'failed'; carriage: string; message: MessageKey };

const geoMessage: Record<string, MessageKey> = {
  denied: 'find.deniedBody',
  unsupported: 'find.unsupportedBody',
  timeout: 'find.timeoutBody',
  outside: 'find.outsideBody',
};

export function useCarriageSearch(client: Pick<Client, 'carriage'> = defaultClient, geo: Geo = browserGeo) {
  const [state, setState] = useState<CarriageState>({ phase: 'form' });
  const abort = useRef<AbortController | null>(null);
  useEffect(() => () => abort.current?.abort(), []);

  const fresh = () => {
    abort.current?.abort();
    abort.current = new AbortController();
    return abort.current.signal;
  };

  const search = useCallback(
    async (carriage: string) => {
      const signal = fresh();
      setState({ phase: 'searching', carriage });
      try {
        const res = await client.carriage(carriage, [], signal);
        if (signal.aborted) return;
        if (res.train) navigate(paths.train(res.train.train, carriage));
        else setState({ phase: 'needLocation', carriage });
      } catch (err) {
        if (!signal.aborted) setState({ phase: 'failed', carriage, message: errorKey(err) });
      }
    },
    [client],
  );

  const searchNearby = useCallback(
    async (carriage: string) => {
      const signal = fresh();
      setState({ phase: 'locating', carriage });
      let fixes;
      try {
        fixes = await sample(geo, { want: 2, maxMs: 8000, signal });
      } catch (err) {
        if (!signal.aborted) setState({ phase: 'failed', carriage, message: geoMessage[(err as Error).message] ?? 'error.unknown' });
        return;
      }
      try {
        const res = await client.carriage(carriage, fixes, signal);
        if (signal.aborted) return;
        if (res.train) navigate(paths.train(res.train.train, carriage));
        else setState({ phase: 'notFound', carriage, searched: res.searched });
      } catch (err) {
        if (!signal.aborted) setState({ phase: 'failed', carriage, message: errorKey(err) });
      }
    },
    [client, geo],
  );

  const reset = useCallback(() => {
    abort.current?.abort();
    setState({ phase: 'form' });
  }, []);

  return { state, search, searchNearby, reset };
}

function Status({ text }: { text: string }) {
  return (
    <div className="mt-10" role="status" aria-live="polite">
      <div aria-hidden="true" className="relative h-7">
        <span className="absolute inset-x-0 top-1/2 h-[9px] -translate-y-1/2 rounded-full bg-line-soft" />
        <span className="track-ride absolute top-0 size-7 rounded-full border-[5px] border-paper bg-sun" />
      </div>
      <p className="mt-5 text-lg font-bold">{text}</p>
    </div>
  );
}

export function CarriageView({ state, search, searchNearby, reset }: ReturnType<typeof useCarriageSearch>) {
  const { t } = useI18n();
  const [value, setValue] = useState('');
  const [invalid, setInvalid] = useState(false);
  const id = useId();

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const n = onlyDigits(value);
    if (n.length < 1 || n.length > 6) return setInvalid(true);
    setInvalid(false);
    void search(n);
  };

  if (state.phase === 'form' || state.phase === 'searching') {
    return (
      <Page title={t('carriage.title')} lead={<p>{t('carriage.lead')}</p>}>
        <form className="mt-8" onSubmit={submit} noValidate>
          <label htmlFor={id} className="block text-base font-bold">
            {t('carriage.label')}
          </label>
          <input
            id={id}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            inputMode="numeric"
            autoComplete="off"
            maxLength={8}
            placeholder="9402"
            aria-invalid={invalid}
            aria-describedby={invalid ? `${id}-error` : undefined}
            className={`mt-2 ${numberInput}`}
          />
          {invalid && (
            <p id={`${id}-error`} className="mt-2 font-bold text-late">
              {t('carriage.invalid')}
            </p>
          )}
          <div className="mt-6 flex flex-col gap-3 sm:flex-row">
            <button type="submit" className={primary} disabled={state.phase === 'searching'}>
              {t('carriage.submit')} <span aria-hidden="true">→</span>
            </button>
          </div>
        </form>
        {state.phase === 'searching' && <Status text={t('carriage.searching', { carriage: state.carriage })} />}
        <p className="mt-10">
          <Link to={paths.other} className={textLink}>
            {t('home.otherWays')}
          </Link>
        </p>
      </Page>
    );
  }

  if (state.phase === 'needLocation' || state.phase === 'locating') {
    return (
      <Page compact title={t('carriage.needLocationTitle')} lead={<p>{t('carriage.needLocationBody', { carriage: state.carriage })}</p>}>
        {state.phase === 'locating' ? (
          <Status text={t('carriage.locating')} />
        ) : (
          <div className="mt-10 flex flex-col gap-3 sm:flex-row">
            <button type="button" className={primary} onClick={() => void searchNearby(state.carriage)}>
              {t('carriage.useLocation')} <span aria-hidden="true">→</span>
            </button>
            <button type="button" className={secondary} onClick={reset}>
              {t('carriage.again')}
            </button>
          </div>
        )}
        <p className="mt-6 text-base font-medium text-dim">{t('find.pointPrivate')}</p>
      </Page>
    );
  }

  const title = state.phase === 'notFound' ? t('carriage.notFoundTitle', { carriage: state.carriage }) : t('find.apiTitle');
  const body = state.phase === 'notFound' ? t('carriage.notFoundBody', { count: state.searched }) : t(state.message);
  return (
    <Page compact title={title} lead={<p>{body}</p>}>
      <div className="mt-10 flex flex-col gap-3 sm:flex-row">
        <button type="button" className={primary} onClick={reset}>
          {t('carriage.again')}
        </button>
        <Link to={paths.other} className={secondary}>
          {t('carriage.other')}
        </Link>
      </div>
    </Page>
  );
}

export function Carriage({ client, geo }: { client?: Pick<Client, 'carriage'>; geo?: Geo }) {
  const flow = useCarriageSearch(client, geo);
  return <CarriageView {...flow} />;
}
