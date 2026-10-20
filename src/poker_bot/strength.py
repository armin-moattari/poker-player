"""Estimate how often a hand wins against random opponent hands."""

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
    opponents: int = 1,
) -> float:
    """Monte Carlo share of the pot won against ``opponents`` random hands.

    A pot split between k tied winners counts 1/k for each of them.
    """
    if opponents < 1:
        raise ValueError(f"need at least one opponent, got {opponents}")
    known = set(hole) | set(board)
    if len(known) != len(hole) + len(board):
        raise ValueError("the same card appears twice in hole cards and board")
    if 2 * opponents + 5 - len(board) > 52 - len(known):
        raise ValueError(f"not enough cards left to deal {opponents} opponents")

    rng = rng or random.Random()
    score = 0.0
    for _ in range(iterations):
        deck = [c for c in Deck(rng).deal(52) if c not in known]
        opp_holes = [deck[2 * i : 2 * i + 2] for i in range(opponents)]
        start = 2 * opponents
        runout = list(board) + deck[start : start + 5 - len(board)]
        mine = best_hand(list(hole) + runout)[0]
        opp_values = [best_hand(h + runout)[0] for h in opp_holes]
        best_opp = max(opp_values)
        if mine > best_opp:
            score += 1.0
        elif mine == best_opp:
            score += 1.0 / (opp_values.count(mine) + 1)
    return score / iterations
