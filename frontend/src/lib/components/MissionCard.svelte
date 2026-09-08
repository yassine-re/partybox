<script lang="ts">
  import type { GameMode, Mission } from "$lib/api/types";
  import { GAME_MODES, MISSION_CATEGORIES } from "$lib/game-modes";
  let {
    mission,
    mode,
    busy,
    oncomplete,
  }: { mission: Mission; mode: GameMode; busy: boolean; oncomplete: () => void } = $props();
  let revealed = $state(true);
  const presentation = $derived(GAME_MODES[mode]);
  const visible = $derived(!presentation.canHide || revealed);
</script>

<section class="mission-card">
  <div class="mission-top">
    <span class="eyebrow">{presentation.cardTitle}</span>{#if presentation.canHide}<button
      class="text-button dark-text"
      onclick={() => (revealed = !revealed)}
      >{revealed ? "Masquer" : "Révéler"}
      <span aria-hidden="true">◉</span></button
    >{/if}
  </div>
  <div class="mission-symbol" aria-hidden="true">{presentation.symbol}</div>
  <p class="mission-copy">
    {visible
      ? mission.text
      : "Rien à voir ici. Juste une soirée tout à fait normale."}
  </p>
  <div class="mission-meta">
    <span>{MISSION_CATEGORIES[mission.category] ?? mission.category}</span><span
      >{["Facile", "Intermédiaire", "Corsée"][mission.difficulty - 1]}</span
    ><strong>+{mission.points} PTS</strong>
  </div>
  <button class="button black" disabled={busy || !visible} onclick={oncomplete}
    >{busy ? "Validation…" : presentation.actionLabel}<span aria-hidden="true">↗</span
    ></button
  >
  <p class="mission-note">{presentation.cardNote}</p>
</section>
