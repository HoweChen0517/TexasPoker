import { CardFace } from './CardFace';
import type { Card, PlayerView } from '../types/poker';

type Props = {
  player?: PlayerView;
  isYou: boolean;
  myCards: Card[];
  activeSeat: number;
  seatIndex: number;
  selectedSeat: number | null;
  onSelectSeat: (seat: number) => void;
};

export function Seat({ player, isYou, myCards, activeSeat, seatIndex, selectedSeat, onSelectSeat }: Props) {
  if (!player) {
    return (
      <button className={`seat empty ${selectedSeat === seatIndex ? 'selected' : ''}`} onClick={() => onSelectSeat(seatIndex)}>
        Empty Seat #{seatIndex + 1}
      </button>
    );
  }

  return (
    <button
      className={`seat ${activeSeat === player.seat ? 'active' : ''} ${player.has_folded ? 'folded' : ''} ${selectedSeat === seatIndex ? 'selected' : ''}`}
      onClick={() => onSelectSeat(seatIndex)}
    >
      <div className="seat-head">
        <span className="name">{player.name}</span>
        {activeSeat === player.seat && <span className="tag turn-tag">{isYou ? 'YOUR TURN' : 'TURN'}</span>}
        {isYou && <span className="tag">YOU</span>}
        {player.is_host && <span className="tag">HOST</span>}
        {player.is_dealer && <span className="tag">D</span>}
        {player.is_small_blind && <span className="tag">SB</span>}
        {player.is_big_blind && <span className="tag">BB</span>}
        {player.has_folded && <span className="tag fold-tag">FOLD</span>}
      </div>
      <div className="chips">{player.chips}</div>
      <div className="bet">Bet {player.current_bet}</div>
      <div className="cards-inline">
        {isYou
          ? myCards.map((c, i) => <CardFace key={`${c.suit}${c.rank}-${i}`} card={c} />)
          : player.shown_cards?.length
            ? player.shown_cards.map((c, i) => <CardFace key={`${c.suit}${c.rank}-shown-${i}`} card={c} />)
            : Array.from({ length: Math.min(2, player.cards_count) }).map((_, i) => <CardFace key={`hidden-${i}`} hidden />)}
      </div>
    </button>
  );
}
