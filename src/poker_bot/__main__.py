"""Run a match from the command line: ``python -m poker_bot --hands 100``."""

from __future__ import annotations

import argparse

from poker_bot.bots import EquityBot, RandomBot
from poker_bot.game import play_match


def main(argv: list[str] | None = None) -> None:
    parser = argparse.ArgumentParser(description="Play EquityBot against RandomBot.")
    parser.add_argument("--hands", type=int, default=100)
    parser.add_argument("--stack", type=int, default=200)
    parser.add_argument("--seed", type=int, default=None)
    parser.add_argument("--verbose", action="store_true", help="print every hand")
    args = parser.parse_args(argv)

    bots = [EquityBot(seed=args.seed), RandomBot(seed=args.seed)]
    stacks = play_match(bots, args.hands, starting_stack=args.stack, seed=args.seed,
                        verbose=args.verbose)
    for bot, stack in zip(bots, stacks):
        print(f"{bot.name}: {stack} ({stack - args.stack:+d})")


if __name__ == "__main__":
    main()
