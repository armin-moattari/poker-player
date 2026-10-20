import random

import pytest

from poker_bot.cards import Card, Deck, parse_cards


def test_parse_and_str_round_trip():
    for text in ["As", "Td", "2c", "Kh"]:
        assert str(Card.parse(text)) == text


def test_parse_rejects_garbage():
    for bad in ["", "A", "1s", "Ax", "10h"]:
        with pytest.raises(ValueError):
            Card.parse(bad)


def test_deck_has_52_unique_cards():
    cards = Deck(random.Random(0)).deal(52)
    assert len(set(cards)) == 52


def test_deck_is_repeatable_with_seed():
    assert Deck(random.Random(7)).deal(5) == Deck(random.Random(7)).deal(5)


def test_deal_too_many_raises():
    deck = Deck()
    deck.deal(50)
    with pytest.raises(ValueError):
        deck.deal(3)


def test_parse_cards():
    assert parse_cards("As Kd") == [Card(14, "s"), Card(13, "d")]
