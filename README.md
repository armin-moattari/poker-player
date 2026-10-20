# poker-player

A heads-up no-limit Texas Hold'em engine in plain Python, with bots that play it.

## What's here

| Module | What it does |
| --- | --- |
| `poker_bot/cards.py` | Cards, parsing (`"As"`, `"Td"`), and a seedable deck |
| `poker_bot/evaluator.py` | Ranks any 5 cards and finds the best 5 of 7 |
| `poker_bot/game.py` | Plays a hand (blinds, four betting streets, showdown) and whole matches |
| `poker_bot/strength.py` | Monte Carlo equity against one or more random hands |
| `poker_bot/advisor.py` | Recommends an action for your current spot at the table |
| `poker_bot/bots.py` | `RandomBot` (baseline) and `EquityBot` (plays by hand strength and pot odds) |

## Quick start

```bash
python -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"
pytest
python -m poker_bot --hands 200 --seed 1          # EquityBot vs RandomBot
python -m poker_bot --hands 3 --seed 1 --verbose  # watch each hand
```

## Hand advisor

Pull it up mid-game to get a recommendation for your spot:

```bash
poker-advisor --hand "As Kd" --board "Qs Jh 2c" --pot 40 --to-call 10 --players 3
```

```
Recommendation: CALL, putting in 10 chips
  flop, holding high card
  win share vs random hands: 39% (fair share 33%)
  pot odds: need 20% to call
  why: you need 20% to break even and win about 39%
```

| Option | Meaning |
| --- | --- |
| `--hand` | your two cards |
| `--board` | community cards so far (leave out preflop); this also sets the betting round |
| `--pot` | chips in the middle, including bets made this round |
| `--to-call` | chips you must put in to call (0 if you can check) |
| `--stack` | chips you have behind (optional; caps raises and calls) |
| `--players` | players still in the hand, you included |

It simulates the rest of the hand against random opponent hands, then raises when you win well
over a fair share of the pot, calls when your win share beats the pot odds, and folds otherwise.
It doesn't yet account for position or how your opponents play; those come in later milestones.

## Writing a bot

A bot is any object with a `name` and an `act(view)` method that returns an `Action`:

```python
from poker_bot.game import Action, PlayerView

class AlwaysCall:
    name = "caller"

    def act(self, view: PlayerView) -> Action:
        return Action.call() if view.to_call else Action.check()
```

`PlayerView` holds your hole cards, the board, the pot, both stacks, what it costs to call,
and the legal raise range. Illegal actions are corrected to the nearest legal one, so a
buggy bot can't break a game.
