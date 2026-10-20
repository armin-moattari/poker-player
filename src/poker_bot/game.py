"""Heads-up no-limit Texas Hold'em: one hand, or a match of many hands."""

from __future__ import annotations

import random
from dataclasses import dataclass, field
from enum import Enum
from typing import Protocol, Sequence

from poker_bot.cards import Card, Deck
from poker_bot.evaluator import best_hand

STREETS = ("preflop", "flop", "turn", "river")


class ActionType(Enum):
    FOLD = "fold"
    CHECK = "check"
    CALL = "call"
    RAISE = "raise"


@dataclass(frozen=True)
class Action:
    """A bot's decision. For RAISE, ``amount`` is the total bet to raise to this street."""

    type: ActionType
    amount: int = 0

    @classmethod
    def fold(cls) -> Action:
        return cls(ActionType.FOLD)

    @classmethod
    def check(cls) -> Action:
        return cls(ActionType.CHECK)

    @classmethod
    def call(cls) -> Action:
        return cls(ActionType.CALL)

    @classmethod
    def raise_to(cls, amount: int) -> Action:
        return cls(ActionType.RAISE, amount)


@dataclass(frozen=True)
class PlayerView:
    """Everything a bot may see when it is asked to act."""

    hole: tuple[Card, Card]
    board: tuple[Card, ...]
    street: str
    pot: int  # chips already in the pot from earlier streets
    my_bet: int  # what I have put in on this street
    opp_bet: int  # what the opponent has put in on this street
    my_stack: int  # chips I have behind
    opp_stack: int
    to_call: int
    min_raise_to: int
    max_raise_to: int
    is_button: bool
    big_blind: int

    @property
    def can_raise(self) -> bool:
        return self.max_raise_to > self.opp_bet


class Bot(Protocol):
    name: str

    def act(self, view: PlayerView) -> Action: ...


@dataclass
class HandResult:
    stacks: list[int]  # stacks after the hand
    winners: list[int]  # seat indexes that won chips at the end
    showdown: bool
    log: list[str] = field(default_factory=list)


def _legalize(action: Action, view: PlayerView) -> Action:
    """Turn any action into the closest legal one, so a buggy bot can't break the game."""
    if action.type is ActionType.RAISE:
        if not view.can_raise:
            return Action.call() if view.to_call else Action.check()
        return Action.raise_to(min(max(action.amount, view.min_raise_to), view.max_raise_to))
    if action.type is ActionType.CHECK and view.to_call:
        return Action.fold()
    if action.type is ActionType.CALL and not view.to_call:
        return Action.check()
    if action.type is ActionType.FOLD and not view.to_call:
        return Action.check()  # never fold when checking is free
    return action


def play_hand(
    bots: Sequence[Bot],
    stacks: Sequence[int],
    button: int,
    small_blind: int,
    big_blind: int,
    rng: random.Random | None = None,
) -> HandResult:
    """Play one hand between two bots. Seat ``button`` posts the small blind."""
    if len(bots) != 2 or len(stacks) != 2:
        raise ValueError("play_hand is heads-up: exactly two bots and two stacks")
    if min(stacks) <= 0:
        raise ValueError("both players need chips to play a hand")

    deck = Deck(rng)
    stacks = list(stacks)
    holes = [tuple(deck.deal(2)), tuple(deck.deal(2))]
    board: list[Card] = []
    bets = [0, 0]
    pot = 0
    folded: int | None = None
    log = [f"{bots[i].name} dealt {holes[i][0]} {holes[i][1]}" for i in range(2)]

    def put_in(seat: int, amount: int) -> None:
        amount = min(amount, stacks[seat])
        stacks[seat] -= amount
        bets[seat] += amount

    bb_seat = 1 - button
    put_in(button, small_blind)
    put_in(bb_seat, big_blind)
    log.append(f"{bots[button].name} posts {bets[button]}, {bots[bb_seat].name} posts {bets[bb_seat]}")

    for street in STREETS:
        if street == "flop":
            board += deck.deal(3)
        elif street in ("turn", "river"):
            board += deck.deal(1)
        if street != "preflop":
            log.append(f"{street}: {' '.join(map(str, board))}")

        acted = [False, False]
        last_raise = big_blind
        seat = button if street == "preflop" else bb_seat

        def needs_to_act(p: int) -> bool:
            o = 1 - p
            if stacks[p] == 0:
                return False
            return bets[p] < bets[o] or (not acted[p] and stacks[o] > 0)

        while folded is None:
            opp = 1 - seat
            if not needs_to_act(seat):
                if not needs_to_act(opp):
                    break
                seat = opp
                continue

            max_raise_to = min(stacks[seat] + bets[seat], stacks[opp] + bets[opp])
            view = PlayerView(
                hole=holes[seat],
                board=tuple(board),
                street=street,
                pot=pot,
                my_bet=bets[seat],
                opp_bet=bets[opp],
                my_stack=stacks[seat],
                opp_stack=stacks[opp],
                to_call=min(bets[opp] - bets[seat], stacks[seat]),
                min_raise_to=min(bets[opp] + last_raise, max_raise_to),
                max_raise_to=max_raise_to,
                is_button=seat == button,
                big_blind=big_blind,
            )
            action = _legalize(bots[seat].act(view), view)
            acted[seat] = True
            name = bots[seat].name

            if action.type is ActionType.FOLD:
                folded = seat
                log.append(f"{name} folds")
            elif action.type is ActionType.CHECK:
                log.append(f"{name} checks")
            elif action.type is ActionType.CALL:
                put_in(seat, view.to_call)
                log.append(f"{name} calls {view.to_call}")
            else:
                last_raise = max(last_raise, action.amount - bets[opp])
                put_in(seat, action.amount - bets[seat])
                acted[opp] = False
                log.append(f"{name} raises to {action.amount}")
            seat = opp

        # Return any uncalled chips (only happens when a short stack posts a blind all-in).
        high = 0 if bets[0] > bets[1] else 1
        refund = bets[high] - bets[1 - high]
        stacks[high] += refund
        pot += bets[0] + bets[1] - refund
        bets = [0, 0]

        if folded is not None:
            break

    if folded is not None:
        winner = 1 - folded
        stacks[winner] += pot
        log.append(f"{bots[winner].name} wins {pot}")
        return HandResult(stacks, [winner], showdown=False, log=log)

    while len(board) < 5:  # both players all-in before the river: run out the board
        board += deck.deal(1)
    values = [best_hand(list(holes[i]) + board)[0] for i in range(2)]
    if values[0] == values[1]:
        winners = [0, 1]
        share, odd_chip = divmod(pot, 2)
        stacks[0] += share
        stacks[1] += share
        stacks[bb_seat] += odd_chip  # odd chip goes to the player out of position
    else:
        winners = [0 if values[0] > values[1] else 1]
        stacks[winners[0]] += pot
    for i in range(2):
        log.append(f"{bots[i].name} shows {holes[i][0]} {holes[i][1]}: {values[i][0].name.lower()}")
    log.append(" and ".join(bots[w].name for w in winners) + f" win {pot}")
    return HandResult(stacks, winners, showdown=True, log=log)


def play_match(
    bots: Sequence[Bot],
    hands: int,
    starting_stack: int = 200,
    small_blind: int = 1,
    big_blind: int = 2,
    seed: int | None = None,
    verbose: bool = False,
) -> list[int]:
    """Play up to ``hands`` hands, alternating the button, and return the final stacks."""
    rng = random.Random(seed)
    stacks = [starting_stack, starting_stack]
    for hand_no in range(hands):
        if min(stacks) == 0:
            break
        result = play_hand(bots, stacks, hand_no % 2, small_blind, big_blind, rng)
        stacks = result.stacks
        if verbose:
            print(f"--- hand {hand_no + 1} ---")
            print("\n".join(result.log))
    return stacks
