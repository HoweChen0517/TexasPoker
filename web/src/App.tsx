import { useEffect, useMemo, useRef, useState } from 'react';
import { CardFace } from './components/CardFace';
import { Seat } from './components/Seat';
import { SpinDial } from './components/SpinDial';
import { useAudioFeedback } from './hooks/useAudioFeedback';
import { useHapticFeedback } from './hooks/useHapticFeedback';
import { usePokerSocket } from './hooks/usePokerSocket';
import type { PlayerView, Snapshot } from './types/poker';

const apiBase = (import.meta.env.VITE_API_URL as string | undefined) || `http://${window.location.hostname}:8080`;

type LoginState = {
  room: string;
  user: string;
  name: string;
  buyIn: number;
};

const LOGIN_STORAGE_KEY = 'texaspoker.login';

function randomUserID() {
  return `u${Math.floor(Math.random() * 100000)}`;
}

function defaultLoginState(): LoginState {
  return {
    room: 'main',
    user: randomUserID(),
    name: '',
    buyIn: 2000
  };
}

function readStoredLogin(): LoginState | null {
  try {
    const raw = window.localStorage.getItem(LOGIN_STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<LoginState>;
    if (!parsed.user) return null;
    return {
      room: parsed.room || 'main',
      user: parsed.user,
      name: parsed.name || '',
      buyIn: Math.max(100, Number(parsed.buyIn) || 2000)
    };
  } catch {
    return null;
  }
}

function readLoginFromLocation(): Partial<LoginState> {
  const params = new URLSearchParams(window.location.search);
  const buyIn = Number(params.get('buyin'));
  return {
    room: params.get('room') || undefined,
    user: params.get('user') || undefined,
    name: params.get('name') || undefined,
    buyIn: Number.isFinite(buyIn) && buyIn > 0 ? buyIn : undefined
  };
}

function resolveInitialLogin(): LoginState {
  const stored = readStoredLogin();
  const defaults = defaultLoginState();
  const fromURL = readLoginFromLocation();
  return {
    room: fromURL.room || stored?.room || defaults.room,
    user: fromURL.user || stored?.user || defaults.user,
    name: fromURL.name || stored?.name || defaults.name,
    buyIn: Math.max(100, fromURL.buyIn || stored?.buyIn || defaults.buyIn)
  };
}

function persistLogin(login: LoginState) {
  const normalized = {
    room: login.room || 'main',
    user: login.user || randomUserID(),
    name: login.name || login.user,
    buyIn: Math.max(100, login.buyIn)
  };
  window.localStorage.setItem(LOGIN_STORAGE_KEY, JSON.stringify(normalized))
  const params = new URLSearchParams(window.location.search);
  params.set('room', normalized.room);
  params.set('user', normalized.user);
  params.set('name', normalized.name);
  params.set('buyin', String(normalized.buyIn));
  window.history.replaceState(null, '', `${window.location.pathname}?${params.toString()}`);
}

function clearPersistedLogin() {
  window.localStorage.removeItem(LOGIN_STORAGE_KEY);
  window.history.replaceState(null, '', window.location.pathname);
}

function seatOrder(players: PlayerView[]) {
  const slots: Array<PlayerView | undefined> = Array.from({ length: 9 }, () => undefined);
  players.forEach((p) => {
    if (p.seat >= 0 && p.seat < 9) slots[p.seat] = p;
  });
  return slots;
}

function phaseLabel(phase?: string) {
  switch (phase) {
    case 'preflop':
      return 'Preflop';
    case 'flop':
      return 'Flop';
    case 'turn':
      return 'Turn';
    case 'river':
      return 'River';
    case 'showdown':
      return 'Showdown';
    case 'complete':
      return 'Settlement';
    default:
      return 'Waiting';
  }
}

function getMinBet(snapshot: Snapshot | null, me: PlayerView | undefined) {
  if (!snapshot || !me) return 0;
  const callCost = Math.max(0, snapshot.current_bet - (me.current_bet ?? 0));
  if (callCost === 0) return Math.min(snapshot.blind_big, me.chips);
  return Math.min(callCost, me.chips);
}

function clampBet(value: number, snapshot: Snapshot | null, me: PlayerView | undefined) {
  if (!snapshot || !me) return Math.max(0, Math.floor(value));
  const chips = Math.max(0, me.chips);
  if (chips === 0) return 0;

  const callCost = Math.max(0, snapshot.current_bet - (me.current_bet ?? 0));
  let next = Math.min(chips, Math.max(0, Math.floor(value)));
  if (callCost === 0) {
    return Math.min(chips, Math.max(next, snapshot.blind_big));
  }

  if (next <= callCost) return Math.min(callCost, chips);
  const minRaiseCommit = callCost + snapshot.min_raise;
  if (chips < minRaiseCommit) return Math.min(callCost, chips);
  if (next < minRaiseCommit) return minRaiseCommit;
  return next;
}

export default function App() {
  const [login, setLogin] = useState<LoginState | null>(null);

  const handleLogin = (next: LoginState) => {
    persistLogin(next);
    setLogin(next);
  };

  const handleLeave = () => {
    setLogin(null);
  };

  if (!login) {
    return <LoginView onSubmit={handleLogin} />;
  }
  return <TableView login={login} onLeave={handleLeave} onResetIdentity={() => {
    clearPersistedLogin();
    setLogin(null);
  }} />;
}

function LoginView({ onSubmit }: { onSubmit: (s: LoginState) => void }) {
  const initial = useMemo(() => resolveInitialLogin(), []);
  const [room, setRoom] = useState(initial.room);
  const [user, setUser] = useState(initial.user);
  const [name, setName] = useState(initial.name);
  const [buyIn, setBuyIn] = useState(initial.buyIn);

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
        <button className="primary" type="submit">Enter Table</button>
        <button
          className="secondary"
          type="button"
          onClick={() => {
            clearPersistedLogin();
            const reset = defaultLoginState();
            setRoom(reset.room);
            setUser(reset.user);
            setName(reset.name);
            setBuyIn(reset.buyIn);
          }}
        >
          Clear Saved Identity
        </button>
      </form>
    </div>
  );
}

function TableView({
  login,
  onLeave,
  onResetIdentity
}: {
  login: LoginState;
  onLeave: () => void;
  onResetIdentity: () => void;
}) {
  const [betAmount, setBetAmount] = useState(40);
  const [startMode, setStartMode] = useState<'classic' | 'short'>('classic');
  const [selectedSeat, setSelectedSeat] = useState<number | null>(null);
  const [phasePop, setPhasePop] = useState('');
  const phaseRef = useRef<string>('');
  const betHandRef = useRef<number | null>(null);
  const audioCtxRef = useRef<AudioContext | null>(null);
  const dialAudio = useAudioFeedback();
  const haptics = useHapticFeedback();
  const { state, snapshot, error, send } = usePokerSocket(apiBase, login.room, login.user, login.name, login.buyIn);

  const seats = useMemo(() => seatOrder(snapshot?.players ?? []), [snapshot]);
  const me = useMemo(() => (snapshot?.players ?? []).find((p) => p.user_id === login.user), [snapshot, login.user]);
  const spectators = useMemo(() => (snapshot?.players ?? []).filter((p) => p.is_spectator), [snapshot]);
  const selectedPlayer = useMemo(() => (selectedSeat === null ? null : seats[selectedSeat] ?? null), [selectedSeat, seats]);
  const actingPlayer = useMemo(
    () => (snapshot?.players ?? []).find((p) => p.seat === snapshot?.acting_seat) || null,
    [snapshot]
  );
  const hostName = useMemo(
    () => (snapshot?.players ?? []).find((p) => p.user_id === snapshot?.host_user_id)?.name || snapshot?.host_user_id || '-',
    [snapshot]
  );
  const isHost = me?.is_host ?? false;
  const isYourTurn = Boolean(
    snapshot &&
      me &&
      !me.is_spectator &&
      !me.has_folded &&
      !me.is_all_in &&
      me.seat === snapshot.acting_seat &&
      snapshot.phase !== 'complete' &&
      snapshot.phase !== 'waiting'
  );
  const amount = betAmount;
  const hasValidAmount = Number.isFinite(amount) && amount > 0;
  const callCost = Math.max(0, (snapshot?.current_bet ?? 0) - (me?.current_bet ?? 0));
  const minBet = getMinBet(snapshot, me);
  const maxBet = Math.max(0, me?.chips ?? 0);
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
    if (!isYourTurn) {
      return { fold: true, check: true, call: true, allIn: true, bet: true, raise: true };
    }

    const canAllIn = me.chips > 0;
    const canFold = callCost > 0;
    const canCall = callCost > 0 && me.chips >= callCost;
    const canCheck = callCost === 0;
    const canBet = hasValidAmount && snapshot.current_bet === 0 && amount >= snapshot.blind_big && amount <= me.chips;
    const raiseBy = amount - callCost;
    const canRaise = hasValidAmount && snapshot.current_bet > 0 && amount <= me.chips && raiseBy >= snapshot.min_raise;

    return {
      fold: !canFold,
      check: !canCheck,
      call: !canCall,
      allIn: !canAllIn,
      bet: !canBet,
      raise: !canRaise
    };
  }, [snapshot, me, amount, hasValidAmount, isYourTurn, callCost]);
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

  useEffect(() => {
    if (!snapshot || !me) return;
    setBetAmount((current) => {
      if (betHandRef.current !== snapshot.hand_id) {
        betHandRef.current = snapshot.hand_id;
        return clampBet(minBet, snapshot, me);
      }
      return clampBet(current || minBet, snapshot, me);
    });
  }, [snapshot?.hand_id, snapshot?.phase, snapshot?.current_bet, snapshot?.min_raise, snapshot?.blind_big, me?.chips, me?.current_bet, minBet, snapshot, me]);

  const clickSeat = async (seat: number) => {
    setSelectedSeat(seat);
  };

  const confirmState = useMemo(() => {
    if (!snapshot || !me || !isYourTurn || !hasValidAmount) {
      return { disabled: true, action: 'bet' as 'bet' | 'call' | 'raise' };
    }

    if (snapshot.current_bet === 0) {
      const canBet = amount >= snapshot.blind_big && amount <= me.chips;
      return { disabled: !canBet, action: 'bet' as const };
    }

    if (amount < callCost) {
      return { disabled: true, action: 'raise' as const };
    }

    if (amount === callCost) {
      return { disabled: actionState.call, action: 'call' as const };
    }

    const raiseBy = amount - callCost
    const canRaise = amount > callCost && amount <= me.chips && raiseBy >= snapshot.min_raise
    return { disabled: !canRaise, action: 'raise' as const }
  }, [snapshot, me, isYourTurn, hasValidAmount, amount, callCost, actionState.call]);

  const fillAmountByPot = (ratio: number) => {
    const pot = snapshot?.pot ?? 0;
    if (pot <= 0) return;
    const computed = Math.max(1, Math.floor(pot * ratio));
    setBetAmount(clampBet(computed, snapshot, me));
  };

  const submitWager = async () => {
    if (!snapshot || !hasValidAmount || confirmState.disabled) return;
    if (confirmState.action === 'bet') {
      await send('action', { action: 'bet', amount });
      return;
    }
    if (confirmState.action === 'call') {
      await send('action', { action: 'call' });
      return;
    }
    await send('action', { action: 'raise', amount: amount - callCost });
  };

  return (
    <div className="page">
      <header className="topbar">
        <h1>Texas Poker House</h1>
        <div className="meta">
          <span>room: {login.room}</span>
          <span>user: {login.name}</span>
          <span>uid: {login.user}</span>
          <span>host: {hostName}</span>
          <span>conn: {state}</span>
          <span>mode: {snapshot?.deck_mode ?? 'classic'}</span>
        </div>
      </header>

      <main className="table-wrap">
        <section className="table">
          {phasePop ? <div className="phase-pop">{phasePop}</div> : null}

          <div className="table-sticky">
            <div className={`turn-banner ${isYourTurn ? 'self' : ''}`}>
              <span className="turn-banner-label">{isYourTurn ? 'Act Now' : 'On Move'}</span>
              <strong>
                {snapshot?.phase === 'waiting' || !actingPlayer
                  ? `Waiting for next hand`
                  : isYourTurn
                    ? `Your decision in ${phaseLabel(snapshot?.phase)}`
                    : `${actingPlayer.name} is acting in ${phaseLabel(snapshot?.phase)}`}
              </strong>
            </div>

            <div className="table-hud">
              <div>phase: {snapshot?.phase ?? 'waiting'}</div>
              <div>current bet: {snapshot?.current_bet ?? 0}</div>
              <div>
                blinds: {snapshot?.blind_small ?? 10}/{snapshot?.blind_big ?? 20}
              </div>
            </div>

            <div className="board-pot-row">
              <div className="board">
                {(snapshot?.board ?? []).map((card, i) => (
                  <CardFace key={`${card.suit}${card.rank}-${i}`} card={card} />
                ))}
                {Array.from({ length: Math.max(0, 5 - (snapshot?.board?.length ?? 0)) }).map((_, i) => (
                  <CardFace key={`empty-${i}`} hidden />
                ))}
              </div>
              <div className="pot-inline">
                <span className="pot-label">Pot</span>
                <span className="pot-value">{snapshot?.pot ?? 0}</span>
              </div>
            </div>

            <div className={`action-dock ${isYourTurn ? 'self-turn' : ''}`}>
              <div className="hero-action-row">
                <div className="me-strip">
                  <div className="me-strip-head">
                    <span>{login.name}</span>
                    <span>{me ? `${me.chips} chips` : '--'}</span>
                  </div>
                  <div className="cards-inline me-cards">
                    {(snapshot?.your_cards ?? []).length
                      ? (snapshot?.your_cards ?? []).map((card, i) => <CardFace key={`${card.suit}${card.rank}-me-${i}`} card={card} />)
                      : [0, 1].map((i) => <CardFace key={`me-empty-${i}`} hidden />)}
                  </div>
                </div>

                <div className="primary-actions">
                  <button disabled={actionState.check} onClick={() => send('action', { action: 'check' })}>
                    Check
                  </button>
                  <button className="primary" disabled={actionState.call} onClick={() => send('action', { action: 'call' })}>
                    Call {callCost > 0 ? callCost : ''}
                  </button>
                  <button disabled={actionState.fold} onClick={() => send('action', { action: 'fold' })}>
                    Fold
                  </button>
                  <button disabled={actionState.allIn} onClick={() => send('action', { action: 'all_in' })}>
                    All In
                  </button>
                </div>
              </div>

              <div className="bet-row">
                <SpinDial
                  value={betAmount}
                  min={minBet}
                  max={maxBet}
                  disabled={!isYourTurn || maxBet <= 0}
                  snapValue={(value) => clampBet(value, snapshot, me)}
                  onChange={(value) => setBetAmount(value)}
                  onTick={() => {
                    dialAudio.playTick();
                    haptics.tick();
                  }}
                  onSnap={() => dialAudio.playSnap()}
                />
                <button className="secondary quick-chip" type="button" onClick={() => fillAmountByPot(0.5)}>
                  1/2 Pot
                </button>
                <button className="secondary quick-chip" type="button" onClick={() => fillAmountByPot(1)}>
                  Pot
                </button>
                <button className="primary confirm-chip" disabled={confirmState.disabled} onClick={submitWager}>
                  Confirm
                </button>
              </div>
            </div>
          </div>

          <div className="message">{snapshot?.round_message ?? 'Waiting for players...'}</div>

          <div className="player-pool">
            <div className="table-surface">
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
          </div>

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
          <div className="control-panel">
            {isHost ? (
              <>
                <label>
                  Start Mode
                  <select value={startMode} onChange={(e) => setStartMode(e.target.value as 'classic' | 'short')}>
                    <option value="classic">Classic</option>
                    <option value="short">Short (remove 2-5, Flush &gt; Full House)</option>
                  </select>
                </label>
                <button className="primary" onClick={() => send('start_hand', { mode: startMode })}>Start Hand</button>
                <button onClick={() => send('restart_hand')}>Restart</button>
                <button onClick={() => send('dissolve_room')}>Dissolve Room</button>
              </>
            ) : (
              <div className="host-only">Only HOST can set mode/start/restart.</div>
            )}

            {me?.is_spectator ? (
              <button className="primary" onClick={() => send('join_table', { seat: selectedSeat ?? -1 })}>Join Next Hand</button>
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
            <button disabled={!snapshot?.can_reveal} onClick={() => send('reveal_cards')}>Reveal My Cards</button>

            <p className="error">{error}</p>
            <button
              onClick={async () => {
                await send('leave_room');
                onLeave();
              }}
            >
              Exit Room
            </button>
            <button className="secondary" onClick={onResetIdentity}>
              Switch Identity
            </button>
          </div>

        </aside>
      </main>
    </div>
  );
}
