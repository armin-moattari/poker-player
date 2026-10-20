"""Estimate how often a hand wins against one random opponent hand."""

from __future__ import annotations

import random
from typing import Sequence

from poker_bot.cards import Card, Deck
from poker_bot.evaluator import best_hand


def equity(
    hole: Sequence[Card],
    board: Sequence[Card],
    iterations: int = 300,
    rng: random.Random | None = None,
) -> float:
    """Monte Carlo share of the pot won (ties count half) versus a random hand."""
    rng = rng or random.Random()
    known = set(hole) | set(board)
    score = 0.0
    for _ in range(iterations):
        deck = [c for c in Deck(rng).deal(52) if c not in known]
        opp = deck[:2]
        runout = list(board) + deck[2 : 2 + 5 - len(board)]
        mine = best_hand(list(hole) + runout)[0]
        theirs = best_hand(opp + runout)[0]
        score += 1.0 if mine > theirs else 0.5 if mine == theirs else 0.0
    return score / iterations
