# poker-player

A heads-up no-limit Texas Hold'em engine in plain Python, with bots that play it.

## What's here

| Module | What it does |
| --- | --- |
| `poker_bot/cards.py` | Cards, parsing (`"As"`, `"Td"`), and a seedable deck |
| `poker_bot/evaluator.py` | Ranks any 5 cards and finds the best 5 of 7 |
| `poker_bot/game.py` | Plays a hand (blinds, four betting streets, showdown) and whole matches |
| `poker_bot/strength.py` | Monte Carlo equity against a random hand |
| `poker_bot/bots.py` | `RandomBot` (baseline) and `EquityBot` (plays by hand strength and pot odds) |

## Quick start

```bash
python -m venv .venv && source .venv/bin/activate
pip install -e ".[dev]"
pytest
python -m poker_bot --hands 200 --seed 1          # EquityBot vs RandomBot
python -m poker_bot --hands 3 --seed 1 --verbose  # watch each hand
```

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
