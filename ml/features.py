"""Leakage-safe feature and target preparation for mission ranking."""

from __future__ import annotations

from collections.abc import Iterable

import pandas as pd


CATEGORICAL_FEATURES = ("mode", "category", "source")
NUMERIC_FEATURES = ("difficulty", "points", "game_size")
FEATURE_COLUMNS = ("mode", "category", "difficulty", "points", "source", "game_size")
AUDIT_COLUMNS = ("assignment_id", "game_id", "mission_id", "player_id")
FORBIDDEN_FEATURES = (
    "rating",
    "liked",
    "completed_at",
    "duration_seconds",
    *AUDIT_COLUMNS,
)


def assert_no_leakage(columns: Iterable[str] = FEATURE_COLUMNS) -> None:
    leaked = sorted(set(columns).intersection(FORBIDDEN_FEATURES))
    if leaked:
        raise ValueError(f"Forbidden prediction features: {', '.join(leaked)}")


def prepare_features(frame: pd.DataFrame) -> pd.DataFrame:
    """Return only values known before assignment, with stable dtypes."""
    assert_no_leakage()
    missing = [column for column in FEATURE_COLUMNS if column not in frame]
    if missing:
        raise ValueError(f"Missing feature columns: {', '.join(missing)}")

    result = frame.loc[:, FEATURE_COLUMNS].copy()
    for column in CATEGORICAL_FEATURES:
        result[column] = result[column].astype("string").str.strip()
        if result[column].isna().any() or result[column].eq("").any():
            raise ValueError(f"Invalid categorical values in {column}")
    for column in NUMERIC_FEATURES:
        result[column] = pd.to_numeric(result[column], errors="raise")
        if result[column].isna().any():
            raise ValueError(f"Missing numeric values in {column}")

    if (result["difficulty"] < 1).any():
        raise ValueError("difficulty must be positive")
    if (result["points"] < 0).any():
        raise ValueError("points must be non-negative")
    if (result["game_size"] < 1).any():
        raise ValueError("game_size must be positive")
    return result


def make_target(frame: pd.DataFrame) -> pd.Series:
    """Map positive feedback to 1; neutral and negative feedback to 0."""
    if "rating" not in frame:
        raise ValueError("Missing target column: rating")
    invalid = ~frame["rating"].isin([-1, 0, 1])
    if invalid.any():
        values = sorted(frame.loc[invalid, "rating"].unique().tolist())
        raise ValueError(f"Invalid ratings found: {values}")
    return frame["rating"].eq(1).astype("int8").rename("liked")
