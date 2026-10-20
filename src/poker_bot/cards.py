"""Playing cards and a shuffled deck."""

from __future__ import annotations

import random
from dataclasses import dataclass

RANKS = "23456789TJQKA"
SUITS = "cdhs"  # clubs, diamonds, hearts, spades


@dataclass(frozen=True, order=True)
class Card:
    """A card with rank 2..14 (ace high) and a suit letter."""

    rank: int
    suit: str

    def __post_init__(self) -> None:
        if not 2 <= self.rank <= 14:
            raise ValueError(f"rank must be 2..14, got {self.rank}")
        if self.suit not in SUITS:
            raise ValueError(f"suit must be one of {SUITS!r}, got {self.suit!r}")

    @classmethod
    def parse(cls, text: str) -> Card:
        """Parse a card like ``"As"`` or ``"Td"``."""
        if len(text) != 2 or text[0].upper() not in RANKS:
            raise ValueError(f"not a card: {text!r}")
        return cls(RANKS.index(text[0].upper()) + 2, text[1].lower())

    def __str__(self) -> str:
        return RANKS[self.rank - 2] + self.suit


def parse_cards(text: str) -> list[Card]:
    """Parse space-separated cards, e.g. ``"As Kd 7h"``."""
    return [Card.parse(token) for token in text.split()]


class Deck:
    """A standard 52-card deck. Pass a seeded ``random.Random`` for repeatable deals."""

    def __init__(self, rng: random.Random | None = None) -> None:
        self._cards = [Card(rank, suit) for suit in SUITS for rank in range(2, 15)]
        (rng or random.Random()).shuffle(self._cards)

    def __len__(self) -> int:
        return len(self._cards)

    def deal(self, n: int = 1) -> list[Card]:
        if n > len(self._cards):
            raise ValueError(f"cannot deal {n} cards, only {len(self._cards)} left")
        dealt, self._cards = self._cards[:n], self._cards[n:]
        return dealt
