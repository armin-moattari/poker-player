"""Hand evaluation: rank any 5 cards and pick the best 5 out of 7."""

from __future__ import annotations

from collections import Counter
from enum import IntEnum
from itertools import combinations
from typing import Iterable

from poker_bot.cards import Card


class Category(IntEnum):
    HIGH_CARD = 0
    PAIR = 1
    TWO_PAIR = 2
    THREE_OF_A_KIND = 3
    STRAIGHT = 4
    FLUSH = 5
    FULL_HOUSE = 6
    FOUR_OF_A_KIND = 7
    STRAIGHT_FLUSH = 8


# A hand's value: compare two values with < and > to see which hand wins.
HandValue = tuple[Category, tuple[int, ...]]


def _straight_high(ranks: list[int]) -> int | None:
    """Return the top rank of a straight in five distinct ranks, or None."""
    unique = sorted(set(ranks), reverse=True)
    if len(unique) != 5:
        return None
    if unique[0] - unique[4] == 4:
        return unique[0]
    if unique == [14, 5, 4, 3, 2]:  # the wheel: A-2-3-4-5
        return 5
    return None


def evaluate5(cards: Iterable[Card]) -> HandValue:
    """Rank exactly five cards."""
    cards = list(cards)
    if len(cards) != 5:
        raise ValueError(f"evaluate5 needs 5 cards, got {len(cards)}")

    ranks = [c.rank for c in cards]
    flush = len({c.suit for c in cards}) == 1
    straight = _straight_high(ranks)

    # Group by count, then rank: e.g. a full house K-K-K-4-4 -> [(3, 13), (2, 4)].
    groups = sorted(((n, r) for r, n in Counter(ranks).items()), reverse=True)
    counts = [n for n, _ in groups]
    by_group = tuple(r for _, r in groups)

    if straight and flush:
        return Category.STRAIGHT_FLUSH, (straight,)
    if counts == [4, 1]:
        return Category.FOUR_OF_A_KIND, by_group
    if counts == [3, 2]:
        return Category.FULL_HOUSE, by_group
    if flush:
        return Category.FLUSH, tuple(sorted(ranks, reverse=True))
    if straight:
        return Category.STRAIGHT, (straight,)
    if counts == [3, 1, 1]:
        return Category.THREE_OF_A_KIND, by_group
    if counts == [2, 2, 1]:
        return Category.TWO_PAIR, by_group
    if counts == [2, 1, 1, 1]:
        return Category.PAIR, by_group
    return Category.HIGH_CARD, by_group


def best_hand(cards: Iterable[Card]) -> tuple[HandValue, tuple[Card, ...]]:
    """Return the best five-card value and those five cards, from 5 to 7 cards."""
    cards = list(cards)
    if not 5 <= len(cards) <= 7:
        raise ValueError(f"best_hand needs 5 to 7 cards, got {len(cards)}")
    return max(((evaluate5(combo), combo) for combo in combinations(cards, 5)),
               key=lambda pair: pair[0])
