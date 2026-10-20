import random

from poker_bot.cards import parse_cards
from poker_bot.strength import equity


def test_aces_are_big_favourites_preflop():
    assert 0.8 < equity(parse_cards("As Ah"), [], 500, random.Random(0)) < 0.9


def test_seven_deuce_is_an_underdog():
    assert equity(parse_cards("7c 2d"), [], 500, random.Random(0)) < 0.4


def test_the_nuts_on_the_river_always_wins():
    board = parse_cards("Ts Js Qs 2d 3c")
    assert equity(parse_cards("As Ks"), board, 100, random.Random(0)) == 1.0
