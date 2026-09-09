"""Load and validate PartyBox's supervised mission-feedback dataset."""

from __future__ import annotations

from dataclasses import dataclass
import sys

import pandas as pd
import psycopg
from psycopg import Connection


SUPERVISED_DATASET_QUERY = """
SELECT
    pm.id::text AS assignment_id,
    g.id::text AS game_id,
    p.id::text AS player_id,
    g.mode,
    m.id AS mission_id,
    m.category,
    m.difficulty,
    m.points,
    m.source,
    mf.rating,
    extract(epoch FROM (pm.completed_at - pm.assigned_at))::double precision
        AS duration_seconds,
    (
        SELECT count(*)::integer
        FROM players game_player
        WHERE game_player.game_id = g.id
          AND game_player.created_at <= pm.completed_at
    ) AS game_size,
    pm.completed_at
FROM mission_feedback mf
JOIN player_missions pm ON pm.id = mf.assignment_id
JOIN missions m ON m.id = pm.mission_id
JOIN players p ON p.id = pm.player_id
JOIN games g ON g.id = p.game_id
WHERE pm.status = 'completed' AND pm.completed_at IS NOT NULL
ORDER BY pm.completed_at, pm.id
"""

COVERAGE_QUERY = """
SELECT
    count(*)::integer AS completed,
    count(mf.assignment_id)::integer AS rated
FROM player_missions pm
LEFT JOIN mission_feedback mf ON mf.assignment_id = pm.id
WHERE pm.status = 'completed' AND pm.completed_at IS NOT NULL
"""

EXPECTED_COLUMNS = [
    "assignment_id",
    "game_id",
    "player_id",
    "mode",
    "mission_id",
    "category",
    "difficulty",
    "points",
    "source",
    "rating",
    "duration_seconds",
    "game_size",
    "completed_at",
]


@dataclass(frozen=True)
class DatasetExport:
    frame: pd.DataFrame
    completed: int
    rated: int
    ignored_durations: int

    @property
    def feedback_rate(self) -> float:
        return self.rated / self.completed if self.completed else 0.0


def _query_frame(connection: Connection, query: str) -> pd.DataFrame:
    with connection.cursor() as cursor:
        cursor.execute(query)
        columns = [column.name for column in cursor.description or ()]
        return pd.DataFrame.from_records(cursor.fetchall(), columns=columns)


def _validate(frame: pd.DataFrame) -> tuple[pd.DataFrame, int]:
    missing_columns = [column for column in EXPECTED_COLUMNS if column not in frame]
    if missing_columns:
        raise ValueError(f"Missing dataset columns: {', '.join(missing_columns)}")
    if frame.empty:
        return frame, 0

    invalid_ratings = ~frame["rating"].isin([-1, 0, 1])
    if invalid_ratings.any():
        values = sorted(frame.loc[invalid_ratings, "rating"].unique().tolist())
        raise ValueError(f"Invalid ratings found: {values}")

    invalid_duration = frame["duration_seconds"].isna() | (
        frame["duration_seconds"] < 0
    )
    ignored = int(invalid_duration.sum())
    if ignored:
        print(
            f"Warning: ignored {ignored} row(s) with an impossible duration.",
            file=sys.stderr,
        )
        frame = frame.loc[~invalid_duration].copy()

    essential = [
        "assignment_id",
        "game_id",
        "player_id",
        "mode",
        "mission_id",
        "category",
        "difficulty",
        "points",
        "source",
        "rating",
        "game_size",
        "completed_at",
    ]
    missing_values = frame[essential].isna().sum()
    missing_values = missing_values[missing_values > 0]
    if not missing_values.empty:
        details = ", ".join(
            f"{column}={int(count)}" for column, count in missing_values.items()
        )
        raise ValueError(f"Missing essential values: {details}")

    blank_fields: list[str] = []
    for column in ["assignment_id", "game_id", "player_id", "mode", "category", "source"]:
        if frame[column].astype("string").str.strip().eq("").any():
            blank_fields.append(column)
    if blank_fields:
        raise ValueError(f"Blank essential values: {', '.join(blank_fields)}")

    return frame, ignored


def load_feedback_dataset(database_url: str) -> DatasetExport:
    """Read rated completions and return a validated, export-ready dataset."""
    with psycopg.connect(database_url) as connection:
        frame = _query_frame(connection, SUPERVISED_DATASET_QUERY)
        with connection.cursor() as cursor:
            cursor.execute(COVERAGE_QUERY)
            completed, rated = cursor.fetchone() or (0, 0)

    frame, ignored = _validate(frame)
    return DatasetExport(
        frame=frame,
        completed=int(completed),
        rated=int(rated),
        ignored_durations=ignored,
    )


def format_summary(dataset: DatasetExport) -> str:
    frame = dataset.frame
    counts = frame["rating"].value_counts() if not frame.empty else {}
    lines = [
        f"Rows: {len(frame)}",
        f"Completed missions: {dataset.completed}",
        f"Rated missions: {dataset.rated}",
        f"Feedback rate: {dataset.feedback_rate:.1%}",
        f"Positive: {int(counts.get(1, 0))}",
        f"Neutral: {int(counts.get(0, 0))}",
        f"Negative: {int(counts.get(-1, 0))}",
    ]
    if dataset.ignored_durations:
        lines.append(f"Ignored durations: {dataset.ignored_durations}")

    lines.append("\nModes:")
    if frame.empty:
        lines.append("  No rated missions yet.")
        return "\n".join(lines)
    for mode, count in frame["mode"].value_counts().sort_index().items():
        lines.append(f"  {mode}: {int(count)}")

    for label, group, value in [
        ("Average rating by mode", "mode", "rating"),
        ("Average rating by category", "category", "rating"),
        ("Average rating by difficulty", "difficulty", "rating"),
        ("Average duration by category (seconds)", "category", "duration_seconds"),
    ]:
        lines.append(f"\n{label}:")
        means = frame.groupby(group, dropna=False)[value].mean().sort_index()
        for key, mean in means.items():
            lines.append(f"  {key}: {mean:.2f}")
    return "\n".join(lines)
