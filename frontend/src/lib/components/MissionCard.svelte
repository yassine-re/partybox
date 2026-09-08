<script lang="ts">
  import type { Mission } from "$lib/api/types";
  let {
    mission,
    busy,
    oncomplete,
  }: { mission: Mission; busy: boolean; oncomplete: () => void } = $props();
  let revealed = $state(true);
</script>

<section class="mission-card">
  <div class="mission-top">
    <span class="eyebrow">TA MISSION SECRÈTE</span><button
      class="text-button dark-text"
      onclick={() => (revealed = !revealed)}
      >{revealed ? "Masquer" : "Révéler"}
      <span aria-hidden="true">◉</span></button
    >
  </div>
  <div class="mission-symbol" aria-hidden="true">✳</div>
  <p class="mission-copy">
    {revealed
      ? mission.text
      : "Rien à voir ici. Juste une soirée tout à fait normale."}
  </p>
  <div class="mission-meta">
    <span>{mission.category}</span><span
      >{["Facile", "Intermédiaire", "Corsée"][mission.difficulty - 1]}</span
    ><strong>+{mission.points} PTS</strong>
  </div>
  <button class="button black" disabled={busy || !revealed} onclick={oncomplete}
    >{busy ? "Validation…" : "J’ai réussi"}<span aria-hidden="true">↗</span
    ></button
  >
  <p class="mission-note">Joue le jeu. Valide seulement quand c’est fait.</p>
</section>
