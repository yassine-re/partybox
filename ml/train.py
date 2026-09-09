"""Train and export PartyBox's global contextual mission ranker."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from datetime import datetime, timezone
import os
from pathlib import Path
import sys
from typing import Any

import pandas as pd
from sklearn.compose import ColumnTransformer
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import GroupShuffleSplit
from sklearn.pipeline import Pipeline
from sklearn.preprocessing import OneHotEncoder, StandardScaler

try:
    from .dataset import format_summary, load_feedback_dataset
    from .evaluate import coefficient_associations, evaluate_classifier, format_evaluation
    from .features import CATEGORICAL_FEATURES, FEATURE_COLUMNS, NUMERIC_FEATURES, make_target, prepare_features
    from .model_metadata import (
        ARTIFACT_DIRECTORY,
        DEFAULT_THRESHOLDS,
        RANDOM_STATE,
        TEST_SIZE,
        TrainingThresholds,
        save_artifacts,
        thresholds_dict,
    )
except ImportError:
    from dataset import format_summary, load_feedback_dataset
    from evaluate import coefficient_associations, evaluate_classifier, format_evaluation
    from features import CATEGORICAL_FEATURES, FEATURE_COLUMNS, NUMERIC_FEATURES, make_target, prepare_features
    from model_metadata import (
        ARTIFACT_DIRECTORY,
        DEFAULT_THRESHOLDS,
        RANDOM_STATE,
        TEST_SIZE,
        TrainingThresholds,
        save_artifacts,
        thresholds_dict,
    )


@dataclass(frozen=True)
class GroupedSplit:
    train_indices: Any
    test_indices: Any
    test_has_both_classes: bool


@dataclass(frozen=True)
class TrainingOutcome:
    trained: bool
    reasons: tuple[str, ...]
    metadata: dict[str, Any] | None = None
    model: Pipeline | None = None


def minimum_data_reasons(frame: pd.DataFrame, target: pd.Series, thresholds: TrainingThresholds = DEFAULT_THRESHOLDS) -> tuple[str, ...]:
    if "game_id" not in frame:
        return ("missing audit column game_id",)
    positives = int(target.sum())
    negatives = int(len(target) - positives)
    games = int(frame["game_id"].nunique())
    reasons: list[str] = []
    if len(frame) < thresholds.min_rows:
        reasons.append(f"rows={len(frame)} < {thresholds.min_rows}")
    if positives < thresholds.min_positive:
        reasons.append(f"positive={positives} < {thresholds.min_positive}")
    if negatives < thresholds.min_negative:
        reasons.append(f"negative={negatives} < {thresholds.min_negative}")
    if games < thresholds.min_games:
        reasons.append(f"games={games} < {thresholds.min_games}")
    return tuple(reasons)


def grouped_train_test_split(frame: pd.DataFrame, target: pd.Series) -> GroupedSplit:
    """Choose a deterministic game-disjoint split, preferring both classes in test."""
    splitter = GroupShuffleSplit(n_splits=64, test_size=TEST_SIZE, random_state=RANDOM_STATE)
    fallback: GroupedSplit | None = None
    for train_indices, test_indices in splitter.split(frame, target, groups=frame["game_id"]):
        train_classes = target.iloc[train_indices].nunique()
        test_classes = target.iloc[test_indices].nunique()
        if train_classes < 2:
            continue
        candidate = GroupedSplit(train_indices, test_indices, test_classes == 2)
        if test_classes == 2:
            return candidate
        if fallback is None:
            fallback = candidate
    if fallback is not None:
        return fallback
    raise ValueError("No grouped split leaves both target classes in training data")


def build_pipeline() -> Pipeline:
    preprocessing = ColumnTransformer(
        transformers=[
            ("categorical", OneHotEncoder(handle_unknown="ignore", sparse_output=False), list(CATEGORICAL_FEATURES)),
            ("numeric", StandardScaler(), list(NUMERIC_FEATURES)),
        ],
        remainder="drop",
        verbose_feature_names_out=True,
    )
    return Pipeline([
        ("preprocessing", preprocessing),
        ("classifier", LogisticRegression(max_iter=1_000, random_state=RANDOM_STATE)),
    ])


def train_frame(
    frame: pd.DataFrame,
    artifact_directory: Path | None = None,
    thresholds: TrainingThresholds = DEFAULT_THRESHOLDS,
) -> TrainingOutcome:
    features = prepare_features(frame)
    target = make_target(frame)
    reasons = minimum_data_reasons(frame, target, thresholds)
    if reasons:
        return TrainingOutcome(False, reasons)

    split = grouped_train_test_split(frame, target)
    train_features = features.iloc[split.train_indices]
    test_features = features.iloc[split.test_indices]
    train_target = target.iloc[split.train_indices]
    test_target = target.iloc[split.test_indices]
    model = build_pipeline()
    model.fit(train_features, train_target)
    majority_class = int(train_target.mode().iloc[0])
    evaluation = evaluate_classifier(model, test_features, test_target, majority_class)
    associations = coefficient_associations(model)
    metadata: dict[str, Any] = {
        "model_type": "logistic_regression",
        "trained_at": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "rows": int(len(frame)),
        "games": int(frame["game_id"].nunique()),
        "positive": int(target.sum()),
        "negative": int(len(target) - target.sum()),
        "features": list(FEATURE_COLUMNS),
        "target": "liked = (rating == 1)",
        "split": {
            "strategy": "group_shuffle_split_by_game_id",
            "random_state": RANDOM_STATE,
            "test_size": TEST_SIZE,
            "train_rows": int(len(split.train_indices)),
            "test_rows": int(len(split.test_indices)),
            "train_games": int(frame.iloc[split.train_indices]["game_id"].nunique()),
            "test_games": int(frame.iloc[split.test_indices]["game_id"].nunique()),
            "test_has_both_classes": split.test_has_both_classes,
        },
        "thresholds": thresholds_dict(thresholds),
        "evaluation": evaluation,
        "coefficient_associations": associations,
        "scope": "global_contextual_not_personalized",
    }
    if artifact_directory is not None:
        save_artifacts(model, metadata, artifact_directory)
    return TrainingOutcome(True, (), metadata, model)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--artifacts", type=Path, default=ARTIFACT_DIRECTORY)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    database_url = os.environ.get("DATABASE_URL", "").strip()
    if not database_url:
        print("DATABASE_URL is required.", file=sys.stderr)
        return 2
    try:
        dataset = load_feedback_dataset(database_url)
        print(format_summary(dataset))
        outcome = train_frame(dataset.frame, args.artifacts)
    except Exception as error:
        print(f"Training failed: {error}", file=sys.stderr)
        return 1
    if not outcome.trained:
        print("\nTraining skipped: Dataset insuffisant pour entraîner un modèle fiable.")
        for reason in outcome.reasons:
            print(f"- {reason}")
        print("No model artifact was created or replaced.")
        return 0
    assert outcome.metadata is not None
    print("\nHeld-out evaluation (games are disjoint):")
    print(format_evaluation(outcome.metadata["evaluation"]))
    if not outcome.metadata["evaluation"]["beats_baseline"]:
        print("Warning: the model did not beat the majority baseline and must not be enabled in PartyBox.")
    print(f"\nArtifacts written to: {args.artifacts}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
