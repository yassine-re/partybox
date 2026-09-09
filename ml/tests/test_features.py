from __future__ import annotations

import pandas as pd
import pytest

from ml.features import FEATURE_COLUMNS, FORBIDDEN_FEATURES, assert_no_leakage, make_target, prepare_features


def test_target_maps_only_positive_feedback_to_liked() -> None:
    target = make_target(pd.DataFrame({"rating": [-1, 0, 1]}))
    assert target.tolist() == [0, 0, 1]


def test_features_exclude_post_outcome_and_identifier_columns() -> None:
    assert set(FEATURE_COLUMNS).isdisjoint(FORBIDDEN_FEATURES)
    with pytest.raises(ValueError, match="Forbidden"):
        assert_no_leakage([*FEATURE_COLUMNS, "duration_seconds"])


def test_feature_preparation_selects_only_pre_assignment_values() -> None:
    frame = pd.DataFrame([{
        "mode": " secret_missions ", "category": "social", "difficulty": 2,
        "points": 15, "source": "seed", "game_size": 4,
        "rating": 1, "duration_seconds": 12, "player_id": "private",
    }])
    prepared = prepare_features(frame)
    assert prepared.columns.tolist() == list(FEATURE_COLUMNS)
    assert prepared.iloc[0]["mode"] == "secret_missions"
