import random

from poker_bot.bots import EquityBot, RandomBot
from poker_bot.game import Action, PlayerView, play_hand, play_match


class Scripted:
    """A bot that replays a fixed list of actions, then checks or calls."""

    def __init__(self, name, actions=()):
        self.name = name
        self.actions = list(actions)
        self.views = []

    def act(self, view: PlayerView) -> Action:
        self.views.append(view)
        if self.actions:
            return self.actions.pop(0)
        return Action.call() if view.to_call else Action.check()


def test_button_fold_preflop_gives_big_blind_the_small_blind():
    bots = [Scripted("a", [Action.fold()]), Scripted("b")]
    result = play_hand(bots, [100, 100], button=0, small_blind=1, big_blind=2,
                       rng=random.Random(0))
    assert result.stacks == [99, 101]
    assert result.winners == [1]
    assert not result.showdown


def test_button_acts_first_preflop_and_last_after():
    a, b = Scripted("a"), Scripted("b")
    play_hand([a, b], [100, 100], button=0, small_blind=1, big_blind=2,
              rng=random.Random(0))
    assert a.views[0].street == "preflop" and a.views[0].to_call == 1
    assert b.views[0].street == "preflop" and b.views[0].to_call == 0  # big blind's option
    assert b.views[1].street == "flop" and a.views[1].street == "flop"
    assert len(b.views) == len(a.views) == 4


def test_check_down_reaches_showdown_and_conserves_chips():
    result = play_hand([Scripted("a"), Scripted("b")], [100, 100], 0, 1, 2,
                       random.Random(3))
    assert result.showdown
    assert sum(result.stacks) == 200


def test_raise_then_fold_returns_uncalled_chips():
    bots = [Scripted("a", [Action.raise_to(10)]), Scripted("b", [Action.fold()])]
    result = play_hand(bots, [100, 100], 0, 1, 2, random.Random(0))
    assert result.stacks == [102, 98]


def test_raises_are_capped_at_the_shorter_stack():
    bots = [Scripted("a", [Action.raise_to(10_000)]), Scripted("b")]
    result = play_hand(bots, [500, 50], 0, 1, 2, random.Random(1))
    assert sum(result.stacks) == 550
    assert result.showdown
    assert result.stacks in ([550, 0], [450, 100], [500, 50])


def test_short_stack_posting_blind_all_in_gets_uncalled_chips_back():
    bots = [Scripted("a"), Scripted("b")]
    result = play_hand(bots, [1, 100], button=1, small_blind=1, big_blind=2,
                       rng=random.Random(2))
    assert sum(result.stacks) == 101
    assert result.stacks[0] in (0, 2, 1)


def test_illegal_actions_are_fixed_up():
    # Checking when facing a bet becomes a fold; folding when checking is free becomes a check.
    bots = [Scripted("a", [Action.check()]), Scripted("b")]
    result = play_hand(bots, [100, 100], 0, 1, 2, random.Random(0))
    assert result.stacks == [99, 101]

    bots = [Scripted("a"), Scripted("b", [Action.fold()])]
    result = play_hand(bots, [100, 100], 0, 1, 2, random.Random(0))
    assert result.showdown


def test_random_bots_never_create_or_lose_chips():
    rng = random.Random(42)
    bots = [RandomBot("x", seed=1), RandomBot("y", seed=2)]
    stacks = [200, 200]
    for hand in range(500):
        if min(stacks) == 0:
            stacks = [200, 200]
        stacks = play_hand(bots, stacks, hand % 2, 1, 2, rng).stacks
        assert sum(stacks) == 400
        assert min(stacks) >= 0


def test_equity_bot_beats_random_bot():
    bots = [EquityBot(iterations=100, seed=5), RandomBot(seed=5)]
    stacks = play_match(bots, hands=200, seed=5)
    assert stacks[0] > stacks[1]
