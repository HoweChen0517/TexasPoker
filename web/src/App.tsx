import { useMemo, useState } from 'react';
import { CardFace } from './components/CardFace';
import { Seat } from './components/Seat';
import { usePokerSocket } from './hooks/usePokerSocket';
import type { PlayerView } from './types/poker';

const user = new URLSearchParams(window.location.search).get('user') || `u${Math.floor(Math.random() * 1000)}`;
const name = new URLSearchParams(window.location.search).get('name') || user;
const room = new URLSearchParams(window.location.search).get('room') || 'main';
const apiBase = (import.meta.env.VITE_API_URL as string | undefined) || `http://${window.location.hostname}:8080`;

function seatOrder(players: PlayerView[]) {
  const slots: Array<PlayerView | undefined> = Array.from({ length: 9 }, () => undefined);
  players.forEach((p) => {
    if (p.seat >= 0 && p.seat < 9) slots[p.seat] = p;
  });
  return slots;
}

export default function App() {
  const [amount, setAmount] = useState(40);
  const { state, snapshot, error, send } = usePokerSocket(apiBase, room, user, name);

  const seats = useMemo(() => seatOrder(snapshot?.players ?? []), [snapshot]);

  return (
    <div className="page">
      <header className="topbar">
        <h1>Texas Poker Doodle Table</h1>
        <div className="meta">
          <span>room: {room}</span>
          <span>user: {name}</span>
          <span>conn: {state}</span>
        </div>
      </header>

      <main className="table-wrap">
        <section className="table">
          <div className="table-hud">
            <div>phase: {snapshot?.phase ?? 'waiting'}</div>
            <div>pot: {snapshot?.pot ?? 0}</div>
            <div>current bet: {snapshot?.current_bet ?? 0}</div>
            <div>
              blinds: {snapshot?.blind_small ?? 10}/{snapshot?.blind_big ?? 20}
            </div>
          </div>

          <div className="board">
            {(snapshot?.board ?? []).map((card, i) => (
              <CardFace key={`${card.suit}${card.rank}-${i}`} card={card} />
            ))}
            {Array.from({ length: Math.max(0, 5 - (snapshot?.board?.length ?? 0)) }).map((_, i) => (
              <CardFace key={`empty-${i}`} hidden />
            ))}
          </div>

          <div className="seats-grid">
            {seats.map((p, idx) => (
              <Seat
                key={`seat-${idx}`}
                player={p}
                isYou={p?.user_id === user}
                myCards={snapshot?.your_cards ?? []}
                activeSeat={snapshot?.acting_seat ?? -1}
              />
            ))}
          </div>

          <div className="message">{snapshot?.round_message ?? 'Waiting for players...'}</div>

          {snapshot?.winners && snapshot.winners.length > 0 && (
            <div className="winner-box">
              {snapshot.winners.map((w) => (
                <div key={w.user_id}>
                  {w.name} +{w.amount} ({w.hand_tag})
                </div>
              ))}
            </div>
          )}
        </section>

        <aside className="controls">
          <button onClick={() => send('start_hand')}>Start Hand</button>
          <button onClick={() => send('action', { action: 'fold' })}>Fold</button>
          <button onClick={() => send('action', { action: 'check' })}>Check</button>
          <button onClick={() => send('action', { action: 'call' })}>Call</button>
          <button onClick={() => send('action', { action: 'all_in' })}>All In</button>

          <label>
            Raise/Bet
            <input type="number" min={1} value={amount} onChange={(e) => setAmount(Number(e.target.value || 1))} />
          </label>

          <div className="actions-inline">
            <button onClick={() => send('action', { action: 'bet', amount })}>Bet</button>
            <button onClick={() => send('action', { action: 'raise', amount })}>Raise</button>
          </div>

          <p className="error">{error}</p>
        </aside>
      </main>
    </div>
  );
}
