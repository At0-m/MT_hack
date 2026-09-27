import { useEffect, useState } from 'react';
import { api, errorText } from '../../api/client';
import type { Calculation, Schema } from '../../api/types';
import { dateLabel, sameWindow } from '../time/dates';

/** Narrative comes from the same server calculation as the chart and CSV. */
export function Summary({ calculation, blocked }: { calculation: Calculation; blocked: boolean }) {
  const [response, setResponse] = useState<Schema['SummaryResponse']>();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setResponse(undefined); setError('');
    if (blocked) { setLoading(false); return () => controller.abort(); }
    setLoading(true);
    void api.summary(calculation.descriptor, calculation.calculation_id, { kind: 'overall' }, controller.signal)
      .then(result => {
        if (controller.signal.aborted) return;
        if (result.calculation_id !== calculation.calculation_id || result.focus.kind !== 'overall' ||
            !sameWindow(result.window, calculation.descriptor.selection.window)) {
          throw new Error('Summary does not match the confirmed calculation.');
        }
        setResponse(result);
      })
      .catch(e => { if (!controller.signal.aborted) setError(errorText(e)); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [calculation, blocked, retry]);
  return <section className="server-summary" aria-label={'\u0421\u0432\u043e\u0434\u043a\u0430 \u043f\u0440\u043e\u0433\u043d\u043e\u0437\u0430'} aria-busy={loading}>
    <h2>{'\u0421\u0432\u043e\u0434\u043a\u0430 \u043f\u0440\u043e\u0433\u043d\u043e\u0437\u0430'}</h2>
    <p className="summary-period">{dateLabel(calculation.descriptor.selection.window.from, 'dd.MM.yyyy HH:mm')}
      {' \u2014 '}{dateLabel(calculation.descriptor.selection.window.to, 'dd.MM.yyyy HH:mm')}{' \u041c\u0421\u041a, \u0434\u043e \u043f\u0440\u0430\u0432\u043e\u0439 \u0433\u0440\u0430\u043d\u0438\u0446\u044b'}</p>
    {blocked ? <p role="status">{'\u041e\u0436\u0438\u0434\u0430\u0435\u043c \u043f\u043e\u0434\u0442\u0432\u0435\u0440\u0436\u0434\u0451\u043d\u043d\u044b\u0439 \u0440\u0430\u0441\u0447\u0451\u0442.'}</p>
      : loading ? <p role="status">{'\u0417\u0430\u0433\u0440\u0443\u0437\u043a\u0430 \u0441\u0432\u043e\u0434\u043a\u0438\u2026'}</p>
      : error ? <p role="alert">{error}<button onClick={() => setRetry(n => n + 1)}>{'\u041f\u043e\u0432\u0442\u043e\u0440\u0438\u0442\u044c'}</button></p>
      : response?.status === 'ready' ? <p>{response.text}</p>
      : <p role="status">{'\u0414\u043b\u044f \u044d\u0442\u043e\u0433\u043e \u0440\u0430\u0441\u0447\u0451\u0442\u0430 \u0442\u0435\u043a\u0441\u0442\u043e\u0432\u0430\u044f \u0441\u0432\u043e\u0434\u043a\u0430 \u043d\u0435\u0434\u043e\u0441\u0442\u0443\u043f\u043d\u0430.'}</p>}
  </section>;
}
