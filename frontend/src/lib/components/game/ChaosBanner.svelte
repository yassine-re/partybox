<script lang="ts">
  import type { ChaosState } from "$lib/api/types";

  let { state }: { state: ChaosState } = $props();
  const event = $derived(state.event);
</script>

<section class="chaos-banner" class:active={state.active} aria-live="polite">
  {#if event?.type === "double_trouble"}
    <div class="chaos-symbol" aria-hidden="true">🔥</div>
    <div>
      <span class="chaos-label">ÉVÉNEMENT CHAOS</span>
      <h2>DOUBLE TROUBLE</h2>
      <p>Les {event.remaining_uses} prochaines missions valent <strong>DOUBLE</strong>.</p>
      <small>{event.remaining_uses} utilisation{event.remaining_uses > 1 ? "s" : ""} restante{event.remaining_uses > 1 ? "s" : ""}</small>
    </div>
  {:else if event?.type === "bounty"}
    <div class="chaos-symbol" aria-hidden="true">🎯</div>
    <div>
      <span class="chaos-label">ÉVÉNEMENT CHAOS</span>
      <h2>BOUNTY</h2>
      <p><strong>{event.target_player_name ?? "Un joueur"}</strong> est la cible.</p>
      <small>Sa prochaine mission rapporte +{event.bonus ?? 100} PTS</small>
    </div>
  {:else if event?.type === "mission_shuffle"}
    <div class="chaos-symbol" aria-hidden="true">🔀</div>
    <div>
      <span class="chaos-label">ÉVÉNEMENT CHAOS</span>
      <h2>MISSION SHUFFLE</h2>
      <p>Nouvelles missions pour tout le monde.</p>
      <small>Regarde ton nouveau défi.</small>
    </div>
  {:else}
    <div class="chaos-symbol dormant" aria-hidden="true">⚡</div>
    <div>
      <span class="chaos-label">INSTABILITÉ EN COURS</span>
      <h2>LE CHAOS SE PRÉPARE</h2>
      <p>{state.progress.completed} / {state.progress.trigger_at} missions avant la prochaine secousse.</p>
    </div>
  {/if}
</section>

<style>
  .chaos-banner {
    display: flex;
    align-items: center;
    gap: 18px;
    margin: 0 0 22px;
    padding: 18px 22px;
    border: 1px solid #6f5d2f;
    border-radius: 12px;
    background: linear-gradient(110deg, #2d291d, #211f24);
  }
  .chaos-banner.active {
    border-color: var(--lime);
    box-shadow: inset 4px 0 var(--pink), 0 0 28px rgb(207 255 89 / 8%);
  }
  .chaos-symbol { flex: 0 0 auto; font-size: 38px; }
  .chaos-symbol.dormant { color: var(--lime); }
  .chaos-label {
    color: var(--lime);
    font: 600 10px monospace;
    letter-spacing: .14em;
  }
  h2 { margin: 3px 0 5px; font-size: clamp(21px, 4vw, 30px); }
  p { margin: 0; color: var(--text); }
  small { display: block; margin-top: 7px; color: var(--muted); }
  strong { color: var(--lime); }
  @media (max-width: 620px) {
    .chaos-banner { align-items: flex-start; padding: 16px; }
    .chaos-symbol { font-size: 30px; }
  }
</style>
