import { CardFace } from './CardFace';
import type { Card, PlayerView } from '../types/poker';

type Props = {
  player?: PlayerView;
  isYou: boolean;
  myCards: Card[];
  activeSeat: number;
};

export function Seat({ player, isYou, myCards, activeSeat }: Props) {
  if (!player) {
    return <div className="seat empty">Empty</div>;
  }

  return (
    <div className={`seat ${activeSeat === player.seat ? 'active' : ''} ${player.has_folded ? 'folded' : ''}`}>
      <div className="seat-head">
        <span className="name">{player.name}</span>
        {player.is_dealer && <span className="tag">D</span>}
        {player.is_small_blind && <span className="tag">SB</span>}
        {player.is_big_blind && <span className="tag">BB</span>}
      </div>
      <div className="chips">{player.chips}</div>
      <div className="bet">Bet {player.current_bet}</div>
      <div className="cards-inline">
        {isYou
          ? myCards.map((c, i) => <CardFace key={`${c.suit}${c.rank}-${i}`} card={c} />)
          : Array.from({ length: Math.min(2, player.cards_count) }).map((_, i) => (
              <CardFace key={`hidden-${i}`} hidden />
            ))}
      </div>
    </div>
  );
}
