"""Artifact paths, training thresholds and portable-model export."""

from __future__ import annotations

from dataclasses import asdict, dataclass
import json
import math
from pathlib import Path
from typing import Any

import joblib

try:
    from .features import CATEGORICAL_FEATURES, FEATURE_COLUMNS, NUMERIC_FEATURES
except ImportError:  # Direct execution from ml/.
    from features import CATEGORICAL_FEATURES, FEATURE_COLUMNS, NUMERIC_FEATURES


ARTIFACT_DIRECTORY = Path(__file__).resolve().parent / "artifacts"
JOBLIB_NAME = "mission_ranker.joblib"
METADATA_NAME = "mission_ranker.metadata.json"
PORTABLE_NAME = "mission_ranker.json"


@dataclass(frozen=True)
class TrainingThresholds:
    min_rows: int = 50
    min_positive: int = 10
    min_negative: int = 10
    min_games: int = 5


DEFAULT_THRESHOLDS = TrainingThresholds()
RANDOM_STATE = 42
TEST_SIZE = 0.2


def thresholds_dict(thresholds: TrainingThresholds) -> dict[str, int]:
    return asdict(thresholds)


def portable_model(pipeline: Any, metadata: dict[str, Any]) -> dict[str, Any]:
    """Export the fitted sklearn transform and binary classifier as plain JSON."""
    preprocessing = pipeline.named_steps["preprocessing"]
    classifier = pipeline.named_steps["classifier"]
    encoder = preprocessing.named_transformers_["categorical"]
    scaler = preprocessing.named_transformers_["numeric"]
    coefficients = classifier.coef_[0]

    categorical: dict[str, dict[str, float]] = {}
    offset = 0
    for feature, categories in zip(CATEGORICAL_FEATURES, encoder.categories_, strict=True):
        categorical[feature] = {
            str(category): float(coefficients[offset + index])
            for index, category in enumerate(categories)
        }
        offset += len(categories)

    numeric: dict[str, dict[str, float]] = {}
    for index, feature in enumerate(NUMERIC_FEATURES):
        scale = float(scaler.scale_[index])
        numeric[feature] = {
            "coefficient": float(coefficients[offset + index]),
            "mean": float(scaler.mean_[index]),
            "scale": scale if scale else 1.0,
        }

    result = {
        "schema_version": 1,
        "model_type": "logistic_regression",
        "feature_order": list(FEATURE_COLUMNS),
        "intercept": float(classifier.intercept_[0]),
        "categorical_features": categorical,
        "numeric_features": numeric,
        "metadata": {
            "trained_at": metadata["trained_at"],
            "rows": metadata["rows"],
            "games": metadata["games"],
            "positive": metadata["positive"],
            "negative": metadata["negative"],
            "beats_baseline": bool(metadata["evaluation"]["beats_baseline"]),
        },
    }
    numbers = [result["intercept"]]
    numbers.extend(value for values in categorical.values() for value in values.values())
    numbers.extend(
        number
        for values in numeric.values()
        for number in (values["coefficient"], values["mean"], values["scale"])
    )
    if not all(math.isfinite(float(number)) for number in numbers):
        raise ValueError("Portable model contains a non-finite value")
    return result


def save_artifacts(pipeline: Any, metadata: dict[str, Any], directory: Path) -> None:
    directory.mkdir(parents=True, exist_ok=True)
    joblib_tmp = directory / f"{JOBLIB_NAME}.tmp"
    metadata_tmp = directory / f"{METADATA_NAME}.tmp"
    portable_tmp = directory / f"{PORTABLE_NAME}.tmp"
    joblib.dump(pipeline, joblib_tmp)
    metadata_tmp.write_text(json.dumps(metadata, indent=2, ensure_ascii=False, allow_nan=False) + "\n")
    portable = portable_model(pipeline, metadata)
    portable_tmp.write_text(json.dumps(portable, indent=2, ensure_ascii=False, allow_nan=False) + "\n")
    joblib_tmp.replace(directory / JOBLIB_NAME)
    metadata_tmp.replace(directory / METADATA_NAME)
    portable_tmp.replace(directory / PORTABLE_NAME)
