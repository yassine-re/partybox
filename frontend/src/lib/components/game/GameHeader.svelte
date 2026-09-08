<script lang="ts">
  import type { Game } from "$lib/api/types";
  import { GAME_MODES } from "$lib/game-modes";

  let { boxId, game }: { boxId: string; game: Game } = $props();
  const mode = $derived(GAME_MODES[game.mode]);
</script>

<div class="game-heading">
  <div>
    <div class="eyebrow">
      <a href="/">ACCUEIL</a><span>/</span> BOX {boxId}<span>/</span> {mode.label.toUpperCase()}
    </div>
    <h1>{game.name}<span class="accent">.</span></h1>
  </div>
  <span class="status-pill"
    ><span class="live-dot" aria-hidden="true"></span>{game.status === "lobby"
      ? "LOBBY OUVERT"
      : game.status === "playing"
        ? "ÇA JOUE"
        : "PARTIE TERMINÉE"}</span
  >
</div>
