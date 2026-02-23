export type Suit = 'S' | 'H' | 'D' | 'C';
export type Rank = '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'T' | 'J' | 'Q' | 'K' | 'A';

export type Card = {
  suit: Suit;
  rank: Rank;
};

export type PlayerView = {
  user_id: string;
  name: string;
  seat: number;
  chips: number;
  current_bet: number;
  has_folded: boolean;
  is_all_in: boolean;
  is_connected: boolean;
  is_spectator: boolean;
  join_next_hand: boolean;
  is_host: boolean;
  cards_count: number;
  shown_cards: Card[];
  is_dealer: boolean;
  is_small_blind: boolean;
  is_big_blind: boolean;
};

export type WinnerView = {
  user_id: string;
  name: string;
  amount: number;
  pot_share: number;
  net_gain: number;
  contribution: number;
  hand_tag: string;
};

export type Snapshot = {
  room_id: string;
  hand_id: number;
  phase: string;
  pot: number;
  current_bet: number;
  min_raise: number;
  blind_small: number;
  blind_big: number;
  deck_mode: 'classic' | 'short';
  host_user_id: string;
  can_reveal: boolean;
  dealer_seat: number;
  acting_seat: number;
  board: Card[];
  players: PlayerView[];
  round_message: string;
  winners: WinnerView[];
  your_cards: Card[];
};
