"""Evaluate a mission ranker against a majority-class baseline."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

import numpy as np
from sklearn.metrics import (
    accuracy_score,
    confusion_matrix,
    f1_score,
    precision_score,
    recall_score,
    roc_auc_score,
)

try:
    from .model_metadata import ARTIFACT_DIRECTORY, METADATA_NAME
except ImportError:
    from model_metadata import ARTIFACT_DIRECTORY, METADATA_NAME


def evaluate_classifier(model: Any, features: Any, target: Any, majority_class: int) -> dict[str, Any]:
    predicted = model.predict(features)
    probabilities = model.predict_proba(features)[:, list(model.classes_).index(1)]
    metrics: dict[str, Any] = {
        "accuracy": float(accuracy_score(target, predicted)),
        "precision": float(precision_score(target, predicted, zero_division=0)),
        "recall": float(recall_score(target, predicted, zero_division=0)),
        "f1": float(f1_score(target, predicted, zero_division=0)),
        "roc_auc": None,
        "confusion_matrix": confusion_matrix(target, predicted, labels=[0, 1]).tolist(),
        "baseline_accuracy": float(accuracy_score(target, np.full(len(target), majority_class))),
    }
    both_classes = len(set(target)) == 2
    if both_classes:
        metrics["roc_auc"] = float(roc_auc_score(target, probabilities))
    metrics["comparison_eligible"] = both_classes
    metrics["beats_baseline"] = both_classes and metrics["accuracy"] > metrics["baseline_accuracy"]
    return metrics


def coefficient_associations(model: Any, limit: int = 8) -> dict[str, list[dict[str, float | str]]]:
    preprocessing = model.named_steps["preprocessing"]
    names = preprocessing.get_feature_names_out()
    coefficients = model.named_steps["classifier"].coef_[0]
    pairs = sorted(zip(names, coefficients, strict=True), key=lambda item: item[1])
    negative = [pair for pair in pairs if pair[1] < 0]
    positive = [pair for pair in pairs if pair[1] > 0]

    def clean(name: str) -> str:
        return name.split("__", 1)[-1]

    return {
        "negative": [{"feature": clean(name), "coefficient": float(value)} for name, value in negative[:limit]],
        "positive": [{"feature": clean(name), "coefficient": float(value)} for name, value in reversed(positive[-limit:])],
    }


def format_evaluation(metrics: dict[str, Any]) -> str:
    lines = [
        f"Baseline accuracy: {metrics['baseline_accuracy']:.3f}",
        f"Model accuracy: {metrics['accuracy']:.3f}",
        f"Precision: {metrics['precision']:.3f}",
        f"Recall: {metrics['recall']:.3f}",
        f"F1: {metrics['f1']:.3f}",
        f"ROC AUC: {metrics['roc_auc']:.3f}" if metrics["roc_auc"] is not None else "ROC AUC: unavailable (one class in test set)",
        f"Confusion matrix [[TN, FP], [FN, TP]]: {metrics['confusion_matrix']}",
        f"Baseline comparison eligible: {'yes' if metrics['comparison_eligible'] else 'no'}",
        f"Beats baseline: {'yes' if metrics['beats_baseline'] else 'no'}",
    ]
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--metadata", type=Path, default=ARTIFACT_DIRECTORY / METADATA_NAME)
    args = parser.parse_args()
    try:
        metadata = json.loads(args.metadata.read_text())
        print(format_evaluation(metadata["evaluation"]))
        print("\nCoefficient associations (not causal effects):")
        print(json.dumps(metadata["coefficient_associations"], indent=2, ensure_ascii=False))
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"Evaluation metadata unavailable: {error}")
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
