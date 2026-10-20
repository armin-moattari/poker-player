import pytest

from poker_bot.cards import parse_cards
from poker_bot.evaluator import Category, best_hand, evaluate5


@pytest.mark.parametrize(
    "hand, category",
    [
        ("As Ks Qs Js Ts", Category.STRAIGHT_FLUSH),
        ("9c 9d 9h 9s 2c", Category.FOUR_OF_A_KIND),
        ("Kc Kd Kh 4s 4c", Category.FULL_HOUSE),
        ("2h 7h 9h Jh Ah", Category.FLUSH),
        ("5c 6d 7h 8s 9c", Category.STRAIGHT),
        ("Ac 2d 3h 4s 5c", Category.STRAIGHT),
        ("7c 7d 7h Ks 2c", Category.THREE_OF_A_KIND),
        ("7c 7d Kh Ks 2c", Category.TWO_PAIR),
        ("7c 7d Qh Ks 2c", Category.PAIR),
        ("7c 9d Qh Ks 2c", Category.HIGH_CARD),
    ],
)
def test_categories(hand, category):
    assert evaluate5(parse_cards(hand))[0] is category


def value(hand):
    return evaluate5(parse_cards(hand))


def test_wheel_is_the_lowest_straight():
    assert value("Ac 2d 3h 4s 5c") < value("2c 3d 4h 5s 6c")


def test_kickers_break_ties():
    assert value("Ac Ad Kh 7s 2c") > value("As Ah Qh 7d 2d")
    assert value("Ac Ad Kh 7s 3c") > value("As Ah Kd 7d 2d")


def test_higher_two_pair_wins():
    assert value("Kc Kd 2h 2s 3c") > value("Qc Qd Jh Js Ac")


def test_full_house_compares_trips_first():
    assert value("3c 3d 3h 2s 2c") > value("2c 2d 2h As Ac")


def test_equal_hands_tie():
    assert value("Ac Kd Qh Js 9c") == value("Ad Kc Qs Jh 9d")


def test_best_hand_picks_best_five_of_seven():
    val, cards = best_hand(parse_cards("2c 3c Ac Kc 9d 9h Qc"))
    assert val[0] is Category.FLUSH
    assert len(cards) == 5


def test_best_hand_finds_straight_on_board():
    val, _ = best_hand(parse_cards("2c 2d 5h 6s 7c 8d 9h"))
    assert val == (Category.STRAIGHT, (9,))


def test_wrong_card_counts_raise():
    with pytest.raises(ValueError):
        evaluate5(parse_cards("As Ks Qs Js"))
    with pytest.raises(ValueError):
        best_hand(parse_cards("As Ks Qs Js"))
