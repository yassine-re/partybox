<script lang="ts">
  import type { ReactionChallenge } from "$lib/api/types";

  let {
    challenge,
    playerId,
  }: {
    challenge: ReactionChallenge;
    playerId: string;
  } = $props();

  const buttonColor = $derived(
    challenge.button_s2_player_id === playerId
      ? "bleu"
      : challenge.button_s3_player_id === playerId
        ? "rouge"
        : null,
  );
  const active = $derived(
    buttonColor !== null && ["awaiting_device", "armed"].includes(challenge.status),
  );
</script>

{#if active}
  <div class="reaction-player-overlay" role="dialog" aria-modal="true" aria-labelledby="player-reaction-title">
    <section class:blue={buttonColor === "bleu"} class:red={buttonColor === "rouge"} class="reaction-player-card">
      <span class="eyebrow">MISSION EN PAUSE</span>
      <span class="reaction-player-flash" aria-hidden="true">⚡</span>
      <h2 id="player-reaction-title">Défi réaction !</h2>
      <p class="reaction-player-lead">Place-toi devant la PartyBox.</p>
      <div class="reaction-button-color">
        <span aria-hidden="true"></span>Bouton {buttonColor}
      </div>
      <p class="reaction-player-instruction">
        N’appuie pas pendant la lumière rouge. Dès que la LED devient verte, appuie le plus vite possible.
      </p>
      <p class="reaction-player-resume">Ta mission reprendra automatiquement juste après.</p>
    </section>
  </div>
{/if}
