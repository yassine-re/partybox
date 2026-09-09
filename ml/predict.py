"""Score and rank mission candidates with a trained local artifact."""

from __future__ import annotations

import argparse
import json
import math
from pathlib import Path
from typing import Any

import joblib
import pandas as pd

try:
    from .features import prepare_features
    from .model_metadata import ARTIFACT_DIRECTORY, JOBLIB_NAME
except ImportError:
    from features import prepare_features
    from model_metadata import ARTIFACT_DIRECTORY, JOBLIB_NAME


def score_missions(model: Any, missions: list[dict[str, Any]], context: dict[str, Any]) -> list[float]:
    if "game_size" not in context:
        raise ValueError("context.game_size is required")
    if not missions:
        return []
    records = [{**mission, "game_size": context["game_size"]} for mission in missions]
    features = prepare_features(pd.DataFrame.from_records(records))
    positive_index = list(model.classes_).index(1)
    return [float(value) for value in model.predict_proba(features)[:, positive_index]]


def rank_missions(model: Any, missions: list[dict[str, Any]], context: dict[str, Any]) -> list[dict[str, Any]]:
    scores = score_missions(model, missions, context)
    ranked = [{**mission, "ml_score": score} for mission, score in zip(missions, scores, strict=True)]
    return sorted(ranked, key=lambda mission: mission["ml_score"], reverse=True)


def score_portable(model: dict[str, Any], candidate: dict[str, Any]) -> float:
    value = float(model["intercept"])
    for feature, categories in model["categorical_features"].items():
        value += float(categories.get(str(candidate[feature]), 0.0))
    for feature, settings in model["numeric_features"].items():
        normalized = (float(candidate[feature]) - float(settings["mean"])) / float(settings["scale"])
        value += float(settings["coefficient"]) * normalized
    if value >= 0:
        return 1.0 / (1.0 + math.exp(-value))
    exponential = math.exp(value)
    return exponential / (1.0 + exponential)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--model", type=Path, default=ARTIFACT_DIRECTORY / JOBLIB_NAME)
    parser.add_argument("--missions", type=Path, required=True, help="JSON array of mission candidates")
    parser.add_argument("--game-size", type=int, required=True)
    args = parser.parse_args()
    try:
        model = joblib.load(args.model)
        missions = json.loads(args.missions.read_text())
        print(json.dumps(rank_missions(model, missions, {"game_size": args.game_size}), indent=2, ensure_ascii=False))
    except (OSError, ValueError, TypeError, KeyError) as error:
        print(f"Prediction failed: {error}")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
