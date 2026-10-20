"""Mid-game advice: given your spot at the table, recommend an action and show why.

Run it from the command line::

    poker-advisor --hand "As Kd" --board "Qs Jh 2c" --pot 40 --to-call 10 --players 3
"""

from __future__ import annotations

import argparse
import random
from dataclasses import dataclass
from typing import Sequence

from poker_bot.cards import RANKS, Card, parse_cards
from poker_bot.evaluator import best_hand
from poker_bot.strength import equity

STREET_BY_BOARD_SIZE = {0: "preflop", 3: "flop", 4: "turn", 5: "river"}

# Raise when equity is this many times a "fair share" of the pot (1 / players in the hand).
STRONG_MULTIPLE = 1.4
# Bet size as a fraction of the pot (after calling).
BET_FRACTION = 0.75


@dataclass(frozen=True)
class Advice:
    action: str  # "fold", "check", "call", "bet" or "raise"
    amount: int  # chips to put in now (0 for fold and check)
    street: str
    hand: str  # e.g. "AKo" preflop, or "two pair" after the flop
    equity: float  # share of the pot you win on average against random hands
    pot_odds: float  # equity you need to break even on a call (0 if nothing to call)
    fair_share: float  # 1 / players in the hand
    reason: str

    def summary(self) -> str:
        verb = self.action.upper() + (f", putting in {self.amount} chips" if self.amount else "")
        lines = [
            f"Recommendation: {verb}",
            f"  {self.street}, holding {self.hand}",
            f"  win share vs random hands: {self.equity:.0%} (fair share {self.fair_share:.0%})",
        ]
        if self.pot_odds:
            lines.append(f"  pot odds: need {self.pot_odds:.0%} to call")
        lines.append(f"  why: {self.reason}")
        return "\n".join(lines)


def describe_hole(hole: Sequence[Card]) -> str:
    """Shorthand like ``AKs``, ``QJo`` or ``77``."""
    high, low = sorted(hole, key=lambda c: c.rank, reverse=True)
    ranks = RANKS[high.rank - 2] + RANKS[low.rank - 2]
    if high.rank == low.rank:
        return ranks
    return ranks + ("s" if high.suit == low.suit else "o")


def advise(
    hole: Sequence[Card],
    board: Sequence[Card] = (),
    pot: int = 0,
    to_call: int = 0,
    stack: int | None = None,
    players: int = 2,
    iterations: int = 1000,
    rng: random.Random | None = None,
) -> Advice:
    """Recommend an action.

    ``pot`` counts every chip already in the middle, including bets made this round.
    ``players`` is how many players are still in the hand, you included.
    ``stack`` is what you have behind; leave it out to ignore stack size.
    """
    if len(hole) != 2:
        raise ValueError(f"need exactly 2 hole cards, got {len(hole)}")
    if len(board) not in STREET_BY_BOARD_SIZE:
        raise ValueError(f"board must have 0, 3, 4 or 5 cards, got {len(board)}")
    if players < 2:
        raise ValueError("need at least 2 players in the hand")
    if pot < 0 or to_call < 0 or (stack is not None and stack <= 0):
        raise ValueError("pot and to-call can't be negative, and stack must be positive")

    street = STREET_BY_BOARD_SIZE[len(board)]
    if board:
        hand = best_hand(list(hole) + list(board))[0][0].name.lower().replace("_", " ")
    else:
        hand = describe_hole(hole)
    win = equity(hole, board, iterations, rng, opponents=players - 1)
    fair_share = 1 / players
    call_cost = to_call if stack is None else min(to_call, stack)
    pot_odds = call_cost / (pot + call_cost) if call_cost else 0.0

    def advice(action: str, amount: int, reason: str) -> Advice:
        return Advice(action, amount, street, hand, win, pot_odds, fair_share, reason)

    strong = win >= min(STRONG_MULTIPLE * fair_share, 0.9)
    can_raise = stack is None or stack > call_cost
    if strong and can_raise:
        size = call_cost + max(round(BET_FRACTION * (pot + call_cost)), 1)
        if stack is not None:
            size = min(size, stack)
        action = "raise" if call_cost else "bet"
        all_in = " (all-in)" if stack is not None and size == stack else ""
        return advice(action, size, f"you win {win:.0%} against {players - 1} opponent(s), "
                      f"well above a fair share of {fair_share:.0%}; build the pot{all_in}")
    if not call_cost:
        return advice("check", 0, "not strong enough to bet for value; see the next card for free")
    if win >= pot_odds:
        return advice("call", call_cost, f"you need {pot_odds:.0%} to break even and win about {win:.0%}")
    return advice("fold", 0, f"calling needs {pot_odds:.0%} but you only win about {win:.0%}")


def main(argv: list[str] | None = None) -> None:
    parser = argparse.ArgumentParser(description="Recommend a poker action for your current spot.")
    parser.add_argument("--hand", required=True, help='your two cards, e.g. "As Kd"')
    parser.add_argument("--board", default="", help='community cards, e.g. "Qs Jh 2c"')
    parser.add_argument("--pot", type=int, default=0, help="chips in the pot, including bets this round")
    parser.add_argument("--to-call", type=int, default=0, help="chips you must put in to call")
    parser.add_argument("--stack", type=int, default=None, help="chips you have behind")
    parser.add_argument("--players", type=int, default=2, help="players still in the hand, including you")
    parser.add_argument("--iterations", type=int, default=1000, help="simulations to run (more = more precise)")
    parser.add_argument("--seed", type=int, default=None, help="random seed for repeatable output")
    args = parser.parse_args(argv)

    try:
        result = advise(
            parse_cards(args.hand),
            parse_cards(args.board),
            pot=args.pot,
            to_call=args.to_call,
            stack=args.stack,
            players=args.players,
            iterations=args.iterations,
            rng=random.Random(args.seed),
        )
    except ValueError as err:
        parser.error(str(err))
    print(result.summary())


if __name__ == "__main__":
    main()
