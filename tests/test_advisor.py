import random

import pytest

from poker_bot.advisor import advise, describe_hole, main
from poker_bot.cards import parse_cards


def run(hole, board="", **kwargs):
    kwargs.setdefault("iterations", 400)
    return advise(parse_cards(hole), parse_cards(board), rng=random.Random(0), **kwargs)


def test_describe_hole():
    assert describe_hole(parse_cards("Kd As")) == "AKo"
    assert describe_hole(parse_cards("Js Ts")) == "JTs"
    assert describe_hole(parse_cards("7c 7d")) == "77"


def test_aces_raise_preflop():
    advice = run("Ah Ad", pot=3, to_call=1, players=2)
    assert advice.action == "raise"
    assert advice.amount > 1
    assert advice.street == "preflop"
    assert advice.hand == "AA"


def test_trash_folds_to_a_big_bet():
    advice = run("7c 2d", pot=10, to_call=10, players=4)
    assert advice.action == "fold"


def test_cheap_call_with_a_draw():
    # Flush draw with a tiny price: calling is clearly profitable.
    advice = run("Ah 5h", "Kh 9h 2c", pot=100, to_call=5, players=2)
    assert advice.action in ("call", "raise")
    assert advice.pot_odds == pytest.approx(5 / 105)


def test_check_when_free_and_not_strong():
    advice = run("8c 3d", "Kh Qs 2c", pot=20, to_call=0, players=3)
    assert advice.action == "check"
    assert advice.amount == 0


def test_bet_the_nuts_on_the_river():
    advice = run("As Ks", "Ts Js Qs 2d 3c", pot=50, to_call=0, players=4)
    assert advice.action == "bet"
    assert advice.equity == 1.0
    assert advice.hand == "straight flush"


def test_more_players_lowers_equity():
    heads_up = run("Qc Qd", players=2).equity
    six_way = run("Qc Qd", players=6).equity
    assert six_way < heads_up


def test_stack_caps_the_raise():
    advice = run("Ah Ad", pot=100, to_call=10, stack=30, players=2)
    assert advice.action == "raise"
    assert advice.amount == 30
    assert "all-in" in advice.reason


def test_short_stack_call_uses_effective_price():
    advice = run("7c 2d", pot=100, to_call=50, stack=5, players=2)
    assert advice.pot_odds == pytest.approx(5 / 105)


@pytest.mark.parametrize(
    "kwargs",
    [
        {"hole": "As", "board": ""},
        {"hole": "As Kd", "board": "Qs Jh"},
        {"hole": "As Kd", "board": "", "players": 1},
        {"hole": "As Kd", "board": "As Jh 2c"},
        {"hole": "As Kd", "board": "", "pot": -1},
    ],
)
def test_bad_input_raises(kwargs):
    with pytest.raises(ValueError):
        run(**kwargs)


def test_cli_prints_a_recommendation(capsys):
    main(["--hand", "As Kd", "--board", "Qs Jh 2c", "--pot", "40", "--to-call", "10",
          "--players", "3", "--iterations", "200", "--seed", "1"])
    out = capsys.readouterr().out
    assert out.startswith("Recommendation:")
    assert "pot odds" in out


def test_cli_reports_bad_cards(capsys):
    with pytest.raises(SystemExit):
        main(["--hand", "As As"])
    assert "same card" in capsys.readouterr().err
