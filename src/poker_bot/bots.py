"""Bots that play heads-up hold'em."""

from __future__ import annotations

import random

from poker_bot.game import Action, PlayerView
from poker_bot.strength import equity


class RandomBot:
    """Picks fold, call/check, or a random-sized raise at random. A baseline to beat."""

    def __init__(self, name: str = "random", seed: int | None = None) -> None:
        self.name = name
        self._rng = random.Random(seed)

    def act(self, view: PlayerView) -> Action:
        roll = self._rng.random()
        if roll < 0.15 and view.to_call:
            return Action.fold()
        if roll < 0.75 or not view.can_raise:
            return Action.call() if view.to_call else Action.check()
        return Action.raise_to(self._rng.randint(view.min_raise_to, view.max_raise_to))


class EquityBot:
    """Plays by hand strength: raises strong hands, calls when the price is right, folds the rest."""

    def __init__(
        self,
        name: str = "equity",
        iterations: int = 200,
        raise_threshold: float = 0.7,
        seed: int | None = None,
    ) -> None:
        self.name = name
        self.iterations = iterations
        self.raise_threshold = raise_threshold
        self._rng = random.Random(seed)

    def act(self, view: PlayerView) -> Action:
        win = equity(view.hole, view.board, self.iterations, self._rng)
        pot_now = view.pot + view.my_bet + view.opp_bet

        if win >= self.raise_threshold and view.can_raise:
            # Bet about the size of the pot.
            return Action.raise_to(view.opp_bet + max(pot_now, view.big_blind))
        if not view.to_call:
            return Action.check()
        pot_odds = view.to_call / (pot_now + view.to_call)
        return Action.call() if win >= pot_odds else Action.fold()
