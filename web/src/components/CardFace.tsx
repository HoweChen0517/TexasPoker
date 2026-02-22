import type { Card } from '../types/poker';

const suitSymbol: Record<string, string> = {
  S: '♠',
  H: '♥',
  D: '♦',
  C: '♣'
};

export function CardFace({ card, hidden = false }: { card?: Card; hidden?: boolean }) {
  if (hidden || !card) {
    return (
      <div className="card card-hidden">
        <span>?</span>
      </div>
    );
  }
  const red = card.suit === 'H' || card.suit === 'D';
  return (
    <div className={`card ${red ? 'red' : ''}`}>
      <span className="rank">{card.rank}</span>
      <span className="suit">{suitSymbol[card.suit]}</span>
    </div>
  );
}
