"""Export rated PartyBox missions to a local CSV file."""

from __future__ import annotations

import argparse
import os
from pathlib import Path
import sys

from dataset import format_summary, load_feedback_dataset


DEFAULT_OUTPUT = Path(__file__).resolve().parent / "data" / "mission_feedback.csv"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        type=Path,
        default=DEFAULT_OUTPUT,
        help=f"CSV destination (default: {DEFAULT_OUTPUT})",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    database_url = os.environ.get("DATABASE_URL", "").strip()
    if not database_url:
        print("DATABASE_URL is required.", file=sys.stderr)
        return 2

    try:
        dataset = load_feedback_dataset(database_url)
        args.output.parent.mkdir(parents=True, exist_ok=True)
        temporary = args.output.with_suffix(args.output.suffix + ".tmp")
        dataset.frame.to_csv(temporary, index=False)
        temporary.replace(args.output)
    except Exception as error:
        print(f"Dataset export failed: {error}", file=sys.stderr)
        return 1

    print(format_summary(dataset))
    print(f"\nExported: {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
