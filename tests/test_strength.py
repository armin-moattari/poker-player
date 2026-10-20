import random

import pytest

from poker_bot.cards import parse_cards
from poker_bot.strength import equity


def test_aces_are_big_favourites_preflop():
    assert 0.8 < equity(parse_cards("As Ah"), [], 500, random.Random(0)) < 0.9


def test_seven_deuce_is_an_underdog():
    assert equity(parse_cards("7c 2d"), [], 500, random.Random(0)) < 0.4


def test_the_nuts_on_the_river_always_wins():
    board = parse_cards("Ts Js Qs 2d 3c")
    assert equity(parse_cards("As Ks"), board, 100, random.Random(0)) == 1.0


def test_equity_drops_with_more_opponents():
    rng = random.Random(1)
    one = equity(parse_cards("Kc Kd"), [], 400, rng, opponents=1)
    four = equity(parse_cards("Kc Kd"), [], 400, rng, opponents=4)
    assert 0.75 < one < 0.9
    assert 0.4 < four < 0.65


def test_board_split_counts_as_share():
    # Royal flush on the board: everyone splits, so each of 3 players gets a third.
    board = parse_cards("As Ks Qs Js Ts")
    assert equity(parse_cards("2c 3d"), board, 50, random.Random(0), opponents=2) == pytest.approx(1 / 3)


def test_rejects_bad_input():
    with pytest.raises(ValueError):
        equity(parse_cards("As Kd"), [], 10, opponents=0)
    with pytest.raises(ValueError):
        equity(parse_cards("As As"), [], 10)
    with pytest.raises(ValueError):
        equity(parse_cards("As Kd"), [], 10, opponents=30)
