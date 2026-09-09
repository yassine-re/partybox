from __future__ import annotations

import json
import math
from pathlib import Path

import joblib
import pandas as pd

from ml.model_metadata import JOBLIB_NAME, METADATA_NAME, PORTABLE_NAME, TrainingThresholds
from ml.predict import rank_missions, score_missions, score_portable
from ml.train import grouped_train_test_split, train_frame


def synthetic_dataset(rows: int = 80, games: int = 10) -> pd.DataFrame:
    records = []
    for index in range(rows):
        game = index % games
        category = "social" if index % 2 == 0 else "physical"
        rating = 1 if category == "social" else (-1 if index % 3 else 0)
        records.append({
            "assignment_id": f"assignment-{index}", "game_id": f"game-{game}",
            "player_id": f"player-{index % 12}", "mission_id": index % 20,
            "mode": ["secret_missions", "treasure_hunt", "chaos"][game % 3],
            "category": category, "difficulty": index % 3 + 1,
            "points": (index % 3 + 1) * 10, "source": "ai" if game % 2 else "seed",
            "game_size": game % 5 + 2, "rating": rating,
            "duration_seconds": 30 + index, "completed_at": "2026-01-01T00:00:00Z",
        })
    return pd.DataFrame.from_records(records)


def test_grouped_split_never_leaks_a_game() -> None:
    frame = synthetic_dataset()
    target = frame["rating"].eq(1).astype(int)
    split = grouped_train_test_split(frame, target)
    train_games = set(frame.iloc[split.train_indices]["game_id"])
    test_games = set(frame.iloc[split.test_indices]["game_id"])
    assert train_games.isdisjoint(test_games)
    repeated = grouped_train_test_split(frame, target)
    assert split.train_indices.tolist() == repeated.train_indices.tolist()
    assert split.test_indices.tolist() == repeated.test_indices.tolist()


def test_small_dataset_is_refused_without_artifact(tmp_path: Path) -> None:
    outcome = train_frame(synthetic_dataset(rows=20, games=4), tmp_path)
    assert not outcome.trained
    assert outcome.reasons
    assert not list(tmp_path.iterdir())


def test_training_artifacts_prediction_ranking_and_unknown_categories(tmp_path: Path) -> None:
    outcome = train_frame(synthetic_dataset(), tmp_path)
    assert outcome.trained and outcome.model is not None and outcome.metadata is not None
    assert {JOBLIB_NAME, METADATA_NAME, PORTABLE_NAME} == {path.name for path in tmp_path.iterdir()}
    loaded = joblib.load(tmp_path / JOBLIB_NAME)
    missions = [
        {"mode": "secret_missions", "category": "never_seen", "difficulty": 1, "points": 10, "source": "seed"},
        {"mode": "secret_missions", "category": "social", "difficulty": 2, "points": 20, "source": "ai"},
    ]
    scores = score_missions(loaded, missions, {"game_size": 5})
    assert len(scores) == 2 and all(0 <= score <= 1 for score in scores)
    portable = json.loads((tmp_path / PORTABLE_NAME).read_text())
    portable_scores = [score_portable(portable, {**mission, "game_size": 5}) for mission in missions]
    assert all(math.isclose(left, right, abs_tol=1e-12) for left, right in zip(scores, portable_scores, strict=True))
    ranked = rank_missions(loaded, missions, {"game_size": 5})
    assert ranked[0]["ml_score"] >= ranked[1]["ml_score"]
    metadata = json.loads((tmp_path / METADATA_NAME).read_text())
    assert "player_id" not in json.dumps(metadata)
    assert metadata["split"]["train_games"] + metadata["split"]["test_games"] == metadata["games"]
    assert {"accuracy", "precision", "recall", "f1", "roc_auc", "confusion_matrix", "baseline_accuracy", "comparison_eligible", "beats_baseline"} <= set(metadata["evaluation"])


def test_portable_fixture_scores_are_shared_with_go() -> None:
    fixtures = Path(__file__).parent / "fixtures"
    model = json.loads((fixtures / "portable_model.json").read_text())
    cases = json.loads((fixtures / "scoring_cases.json").read_text())
    for case in cases:
        assert math.isclose(score_portable(model, case["input"]), case["python_score"], abs_tol=1e-9)


def test_thresholds_can_be_overridden_only_for_focused_tests(tmp_path: Path) -> None:
    thresholds = TrainingThresholds(min_rows=10, min_positive=2, min_negative=2, min_games=3)
    assert train_frame(synthetic_dataset(rows=24, games=6), tmp_path, thresholds).trained
