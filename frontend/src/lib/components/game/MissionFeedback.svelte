<script lang="ts">
  import type { MissionFeedbackRating } from "$lib/api/types";

  let {
    assignmentId,
    busy,
    error = "",
    onsubmit,
    onskip,
  }: {
    assignmentId: string;
    busy: boolean;
    error?: string;
    onsubmit: (rating: MissionFeedbackRating) => void;
    onskip: () => void;
  } = $props();

  const choices: ReadonlyArray<{
    rating: MissionFeedbackRating;
    emoji: string;
    label: string;
  }> = [
    { rating: -1, emoji: "👎", label: "Bof" },
    { rating: 0, emoji: "😐", label: "Ça va" },
    { rating: 1, emoji: "🔥", label: "Fun" },
  ];
</script>

<section
  class="feedback-panel"
  aria-labelledby="mission-feedback-title"
  data-assignment-id={assignmentId}
>
  <span class="eyebrow">AVIS EXPRESS · FACULTATIF</span>
  <h2 id="mission-feedback-title">Cette mission était comment ?</h2>
  <div class="feedback-choices" role="group" aria-label="Noter la mission terminée">
    {#each choices as choice}
      <button
        type="button"
        class="feedback-choice"
        disabled={busy}
        onclick={() => onsubmit(choice.rating)}
      >
        <span aria-hidden="true">{choice.emoji}</span>
        <span class="sr-only">{choice.label}</span>
      </button>
    {/each}
  </div>
  {#if error}<p class="feedback-error" role="status">{error}</p>{/if}
  <button type="button" class="text-button" disabled={busy} onclick={onskip}>
    {busy ? "Envoi…" : "Passer"}
  </button>
</section>

<style>
  .feedback-panel {
    margin-top: 12px;
    padding: 12px 14px;
    border: 1px solid #596840;
    border-radius: 14px;
    background: linear-gradient(145deg, #202719, #171a14);
  }
  h2 {
    margin: 5px 0 9px;
    font-size: 1rem;
    letter-spacing: -0.035em;
  }
  .feedback-choices {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }
  .feedback-choice {
    min-height: 44px;
    padding: 5px 6px;
    border: 1px solid #4a513e;
    border-radius: 9px;
    background: #292e23;
    color: var(--text);
    font-weight: 800;
  }
  .feedback-choice span {
    display: block;
    margin-bottom: 4px;
    font-size: 22px;
  }
  .feedback-choice .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
  .feedback-choice { font-size: 22px; }
  .feedback-choice:hover:enabled,
  .feedback-choice:focus-visible {
    border-color: var(--lime);
    background: #333b28;
  }
  .feedback-error {
    margin: 6px 0 0;
    color: #ffadb9;
    font-size: 12px;
  }
  .text-button {
    display: block;
    margin: 4px auto -4px;
  }

  @media (max-width: 420px) {
    .feedback-panel {
      padding: 11px 12px;
    }
    .feedback-choice {
      min-height: 42px;
      font-size: 22px;
    }
  }
</style>
