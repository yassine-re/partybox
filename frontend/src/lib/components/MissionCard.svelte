<script lang="ts">
  import type { Snippet } from "svelte";
  import type { GameMode, Mission } from "$lib/api/types";
  import { GAME_MODES, MISSION_CATEGORIES } from "$lib/game-modes";
  import { onMount } from "svelte";
  let {
    mission,
    mode,
    busy,
    validationPending = false,
    oncomplete,
    onskip,
    actions,
  }: { mission: Mission; mode: GameMode; busy: boolean; validationPending?: boolean; oncomplete: () => void; onskip?: () => void; actions?: Snippet } = $props();
  let revealed = $state(false);
  let hideTimer: ReturnType<typeof setTimeout> | undefined;
  const presentation = $derived(GAME_MODES[mode]);
  const visible = $derived(!presentation.canHide || revealed);
  function reveal() {
    revealed = true;
    if (hideTimer) clearTimeout(hideTimer);
    if (presentation.canHide) hideTimer = setTimeout(() => (revealed = false), 12_000);
  }
  function toggleReveal() {
    if (revealed) {
      revealed = false;
      if (hideTimer) clearTimeout(hideTimer);
    } else reveal();
  }
  $effect(() => {
    mission.id;
    revealed = false;
    if (hideTimer) clearTimeout(hideTimer);
  });
  onMount(() => {
    const hideWhenBackgrounded = () => {
      if (document.hidden && presentation.canHide) {
        revealed = false;
        if (hideTimer) clearTimeout(hideTimer);
      }
    };
    document.addEventListener("visibilitychange", hideWhenBackgrounded);
    return () => {
      document.removeEventListener("visibilitychange", hideWhenBackgrounded);
      if (hideTimer) clearTimeout(hideTimer);
    };
  });
</script>

<section class="mission-card">
  <div class="mission-top">
    <span class="eyebrow">{presentation.cardTitle}</span>{#if presentation.canHide}<button
      class="text-button dark-text"
      onclick={toggleReveal}
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
  {#if actions}
    {@render actions()}
  {:else}
  <button class="button black" disabled={busy || !visible} onclick={oncomplete}
    >{validationPending ? "En attente de confirmation" : busy ? "Validation…" : presentation.actionLabel}<span aria-hidden="true">↗</span
  ></button
  >
  <p class="mission-note">{validationPending ? "Ta demande a été envoyée. Un autre joueur doit confirmer cette mission." : presentation.cardNote}</p>
  {/if}
  {#if mode === "treasure_hunt" && onskip}
    <button class="text-button dark-text skip-mission" disabled={busy} onclick={onskip}>Impossible à trouver ? Passer cette recherche</button>
  {/if}
</section>

<style>
  .skip-mission { justify-self: center; margin-top: 4px; text-decoration: underline; text-underline-offset: 3px; }
</style>
