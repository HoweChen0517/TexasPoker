import { useEffect, useMemo, useRef, useState } from 'react';
import { CardFace } from './components/CardFace';
import { Seat } from './components/Seat';
import { usePokerSocket } from './hooks/usePokerSocket';
import type { PlayerView } from './types/poker';

const apiBase = (import.meta.env.VITE_API_URL as string | undefined) || `http://${window.location.hostname}:8080`;

type LoginState = {
  room: string;
  user: string;
  name: string;
  buyIn: number;
};

function seatOrder(players: PlayerView[]) {
  const slots: Array<PlayerView | undefined> = Array.from({ length: 9 }, () => undefined);
  players.forEach((p) => {
    if (p.seat >= 0 && p.seat < 9) slots[p.seat] = p;
  });
  return slots;
}

export default function App() {
  const [login, setLogin] = useState<LoginState | null>(null);

  if (!login) {
    return <LoginView onSubmit={setLogin} />;
  }
  return <TableView login={login} onLeave={() => setLogin(null)} />;
}

function LoginView({ onSubmit }: { onSubmit: (s: LoginState) => void }) {
  const params = new URLSearchParams(window.location.search);
  const [room, setRoom] = useState(params.get('room') || 'main');
  const [user, setUser] = useState(params.get('user') || `u${Math.floor(Math.random() * 1000)}`);
  const [name, setName] = useState(params.get('name') || '');
  const [buyIn, setBuyIn] = useState(Number(params.get('buyin') || 2000));

  return (
    <div className="login-page">
      <form
        className="login-card"
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit({ room, user, name: name || user, buyIn: Math.max(100, buyIn) });
        }}
      >
        <h1>Join Poker Room</h1>
        <label>
          Room
          <input value={room} onChange={(e) => setRoom(e.target.value)} />
        </label>
        <label>
          User ID
          <input value={user} onChange={(e) => setUser(e.target.value)} />
        </label>
        <label>
          Nickname
          <input value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label>
          Buy-in Chips
          <input type="number" min={100} value={buyIn} onChange={(e) => setBuyIn(Number(e.target.value || 100))} />
        </label>
        <button type="submit">Enter Table</button>
      </form>
    </div>
  );
}

function TableView({ login, onLeave }: { login: LoginState; onLeave: () => void }) {
  const [amountInput, setAmountInput] = useState('40');
  const [startMode, setStartMode] = useState<'classic' | 'short'>('classic');
  const [selectedSeat, setSelectedSeat] = useState<number | null>(null);
  const [phasePop, setPhasePop] = useState('');
  const phaseRef = useRef<string>('');
  const audioCtxRef = useRef<AudioContext | null>(null);
  const { state, snapshot, error, send } = usePokerSocket(apiBase, login.room, login.user, login.name, login.buyIn);

  const seats = useMemo(() => seatOrder(snapshot?.players ?? []), [snapshot]);
  const me = useMemo(() => (snapshot?.players ?? []).find((p) => p.user_id === login.user), [snapshot, login.user]);
  const spectators = useMemo(() => (snapshot?.players ?? []).filter((p) => p.is_spectator), [snapshot]);
  const selectedPlayer = useMemo(() => (selectedSeat === null ? null : seats[selectedSeat] ?? null), [selectedSeat, seats]);
  const hostName = useMemo(
    () => (snapshot?.players ?? []).find((p) => p.user_id === snapshot?.host_user_id)?.name || snapshot?.host_user_id || '-',
    [snapshot]
  );
  const isHost = me?.is_host ?? false;
  const amount = Number.parseInt(amountInput, 10);
  const hasValidAmount = Number.isFinite(amount) && amount > 0;
  const actionState = useMemo(() => {
    const disabledAll = {
      fold: false,
      check: false,
      call: false,
      allIn: false,
      bet: false,
      raise: false
    };
    if (!snapshot || !me) return disabledAll;
    const isYourTurn =
      !me.is_spectator &&
      !me.has_folded &&
      !me.is_all_in &&
      me.seat === snapshot.acting_seat &&
      snapshot.phase !== 'complete' &&
      snapshot.phase !== 'waiting';
    if (!isYourTurn) {
      return { fold: true, check: true, call: true, allIn: true, bet: true, raise: true };
    }

    const callCost = Math.max(0, snapshot.current_bet - me.current_bet);
    const canAllIn = me.chips > 0;
    const canFold = callCost > 0;
    const canCall = callCost > 0 && me.chips >= callCost;
    const canCheck = callCost === 0;
    const canBet = hasValidAmount && snapshot.current_bet === 0 && me.chips >= Math.max(snapshot.blind_big, amount);
    const needToRaiseBy = snapshot.current_bet > 0 ? callCost + amount : amount;
    const canRaise = hasValidAmount && snapshot.current_bet > 0 && amount >= snapshot.min_raise && me.chips >= needToRaiseBy;

    return {
      fold: !canFold,
      check: !canCheck,
      call: !canCall,
      allIn: !canAllIn,
      bet: !canBet,
      raise: !canRaise
    };
  }, [snapshot, me, amount, hasValidAmount]);
  const canRemoveSelected = Boolean(
    isHost &&
      snapshot &&
      (snapshot.phase === 'waiting' || snapshot.phase === 'complete') &&
      selectedPlayer &&
      selectedPlayer.user_id !== login.user
  );

  useEffect(() => {
    const initAudio = async () => {
      if (!audioCtxRef.current) {
        audioCtxRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
      }
      if (audioCtxRef.current.state === 'suspended') {
        try {
          await audioCtxRef.current.resume();
        } catch {
          // ignored
        }
      }
    };
    const unlock = () => void initAudio();
    window.addEventListener('pointerdown', unlock, { passive: true });
    window.addEventListener('keydown', unlock);
    return () => {
      window.removeEventListener('pointerdown', unlock);
      window.removeEventListener('keydown', unlock);
    };
  }, []);

  useEffect(() => {
    if (!snapshot?.phase) return;
    if (!phaseRef.current) {
      phaseRef.current = snapshot.phase;
      return;
    }
    if (phaseRef.current === snapshot.phase) return;
    phaseRef.current = snapshot.phase;
    const pretty = snapshot.phase.toUpperCase();
    setPhasePop(pretty);
    const timeout = window.setTimeout(() => setPhasePop(''), 850);
    const playCue = async () => {
      try {
        if (!audioCtxRef.current) {
          audioCtxRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
        }
        if (audioCtxRef.current.state === 'suspended') {
          await audioCtxRef.current.resume();
        }
        const audioCtx = audioCtxRef.current;
        const osc = audioCtx.createOscillator();
        const gain = audioCtx.createGain();
        osc.type = 'triangle';
        osc.frequency.value = 640;
        gain.gain.value = 0.001;
        osc.connect(gain);
        gain.connect(audioCtx.destination);
        const now = audioCtx.currentTime;
        gain.gain.exponentialRampToValueAtTime(0.09, now + 0.01);
        gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.16);
        osc.start(now);
        osc.stop(now + 0.16);
      } catch {
        // ignored
      }
    };
    void playCue();

    return () => window.clearTimeout(timeout);
  }, [snapshot?.phase]);

  const clickSeat = async (seat: number) => {
    setSelectedSeat(seat);
  };

  return (
    <div className="page">
      <header className="topbar">
        <h1>Texas Poker House</h1>
        <div className="meta">
          <span>room: {login.room}</span>
          <span>user: {login.name}</span>
          <span>host: {hostName}</span>
          <span>conn: {state}</span>
          <span>mode: {snapshot?.deck_mode ?? 'classic'}</span>
        </div>
      </header>

      <main className="table-wrap">
        <section className="table">
          <div className="table-hud">
            <div>phase: {snapshot?.phase ?? 'waiting'}</div>
            <div>current bet: {snapshot?.current_bet ?? 0}</div>
            <div>
              blinds: {snapshot?.blind_small ?? 10}/{snapshot?.blind_big ?? 20}
            </div>
          </div>

          <div className="pot-highlight">
            <span className="pot-label">POT</span>
            <span className="pot-value">{snapshot?.pot ?? 0}</span>
          </div>
          {phasePop ? <div className="phase-pop">{phasePop}</div> : null}

          <div className="table-surface">
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
                  isYou={p?.user_id === login.user}
                  myCards={snapshot?.your_cards ?? []}
                  activeSeat={snapshot?.acting_seat ?? -1}
                  seatIndex={idx}
                  selectedSeat={selectedSeat}
                  onSelectSeat={clickSeat}
                />
              ))}
            </div>
            <div className="phase-corner">{snapshot?.phase ?? 'waiting'}</div>
          </div>

          <div className="message">{snapshot?.round_message ?? 'Waiting for players...'}</div>

          {snapshot?.winners && snapshot.winners.length > 0 && (
            <div className="winner-box">
              {snapshot.winners.map((w) => (
                <div key={w.user_id}>
                  {w.name} 收池 {w.pot_share} / 净赢 {w.net_gain >= 0 ? `+${w.net_gain}` : w.net_gain} ({w.hand_tag})
                </div>
              ))}
            </div>
          )}

          <div className="spectator-box">
            <strong>Spectators:</strong>
            <div>{spectators.length ? spectators.map((s) => `${s.name}${s.join_next_hand ? ' (next hand)' : ''}`).join(' / ') : 'none'}</div>
          </div>
        </section>

        <aside className="controls">
          {isHost ? (
            <>
              <label>
                Start Mode
                <select value={startMode} onChange={(e) => setStartMode(e.target.value as 'classic' | 'short')}>
                  <option value="classic">Classic</option>
                  <option value="short">Short (remove 2-5, Flush &gt; Full House)</option>
                </select>
              </label>
              <button onClick={() => send('start_hand', { mode: startMode })}>Start Hand</button>
              <button onClick={() => send('restart_hand')}>Restart</button>
              <button onClick={() => send('dissolve_room')}>Dissolve Room</button>
            </>
          ) : (
            <div className="host-only">Only HOST can set mode/start/restart.</div>
          )}

          {me?.is_spectator ? (
            <button onClick={() => send('join_table', { seat: selectedSeat ?? -1 })}>Join Next Hand</button>
          ) : (
            <button disabled={selectedSeat === null} onClick={() => send('set_seat', { seat: selectedSeat ?? -1 })}>
              Change Seat
            </button>
          )}
          <button
            disabled={!canRemoveSelected}
            onClick={() => selectedPlayer && send('remove_player', { user_id: selectedPlayer.user_id })}
          >
            {selectedPlayer ? `Remove ${selectedPlayer.name}` : 'Remove Player'}
          </button>
          <button disabled={!snapshot?.can_reveal} onClick={() => send('reveal_cards')}>
            Reveal My Cards
          </button>

          <button disabled={actionState.fold} onClick={() => send('action', { action: 'fold' })}>
            Fold
          </button>
          <button disabled={actionState.check} onClick={() => send('action', { action: 'check' })}>
            Check
          </button>
          <button disabled={actionState.call} onClick={() => send('action', { action: 'call' })}>
            Call
          </button>
          <button disabled={actionState.allIn} onClick={() => send('action', { action: 'all_in' })}>
            All In
          </button>

          <label>
            Raise/Bet
            <input
              type="text"
              inputMode="numeric"
              placeholder="Enter amount"
              value={amountInput}
              onChange={(e) => setAmountInput(e.target.value.replace(/[^\d]/g, ''))}
            />
          </label>

          <div className="actions-inline">
            <button disabled={actionState.bet || !hasValidAmount} onClick={() => send('action', { action: 'bet', amount })}>
              Bet
            </button>
            <button disabled={actionState.raise || !hasValidAmount} onClick={() => send('action', { action: 'raise', amount })}>
              Raise
            </button>
          </div>

          <p className="error">{error}</p>
          <button
            onClick={async () => {
              await send('leave_room');
              onLeave();
            }}
          >
            Exit Room
          </button>
        </aside>
      </main>
    </div>
  );
}
