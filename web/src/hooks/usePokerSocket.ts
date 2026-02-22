import { useEffect, useMemo, useRef, useState } from 'react';
import type { Snapshot } from '../types/poker';

type ConnState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

export function usePokerSocket(baseUrl: string, room: string, user: string, name: string, buyIn: number) {
  const eventRef = useRef<EventSource | null>(null);
  const [state, setState] = useState<ConnState>('idle');
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [error, setError] = useState<string>('');

  useEffect(() => {
    setState('connecting');
    const url = `${baseUrl}/events?room=${encodeURIComponent(room)}&user=${encodeURIComponent(user)}&name=${encodeURIComponent(name)}&buy_in=${buyIn}`;
    const es = new EventSource(url);
    eventRef.current = es;

    es.onopen = () => setState('open');
    es.onerror = () => {
      setState('error');
      setError('Event stream disconnected');
    };
    es.onmessage = (ev) => {
      const msg = JSON.parse(ev.data) as { type: string; payload: any };
      if (msg.type === 'snapshot') setSnapshot(msg.payload as Snapshot);
      if (msg.type === 'error') setError(msg.payload?.message ?? 'Unknown error');
    };

    return () => {
      es.close();
      setState('closed');
    };
  }, [baseUrl, room, user, name, buyIn]);

  const send = useMemo(
    () => async (type: string, payload: Record<string, unknown> = {}) => {
      const actionUrl = `${baseUrl}/action?room=${encodeURIComponent(room)}&user=${encodeURIComponent(user)}`;
      await fetch(actionUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, payload })
      });
    },
    [baseUrl, room, user]
  );

  return { state, snapshot, error, send };
}
