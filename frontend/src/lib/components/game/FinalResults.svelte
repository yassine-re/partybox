<script lang="ts">
  import type { Game, Player } from "$lib/api/types";
  import { GAME_MODES } from "$lib/game-modes";
  import PlayerList from "../PlayerList.svelte";

  let {
    game,
    players,
    playerId,
    busy,
    onback,
  }: {
    game: Game;
    players: Player[];
    playerId: string;
    busy: boolean;
    onback: () => void;
  } = $props();
  const mode = $derived(GAME_MODES[game.mode]);
  const winners = $derived(
    players.filter((player) => player.score === players[0]?.score),
  );
</script>

<div class="final-layout">
  <section class="final-feature">
    <div class="eyebrow">{mode.finalEyebrow}</div>
    <span class="final-trophy" aria-hidden="true">✳</span>
    <h2>
      Bien joué,<br /><em>{winners.map((player) => player.name).join(" & ")}.</em>
    </h2>
    <p>
      {winners.length > 1
        ? "La victoire se partage ce soir."
        : "La soirée a son champion."}
      {players.reduce((sum, player) => sum + player.completed_missions, 0)} {mode.completedPlural}
      ensemble.
    </p>
    <div class="final-score">
      {players[0]?.score ?? 0}<span>POINTS AU SOMMET</span>
    </div>
    <button class="button primary" onclick={onback} disabled={busy}
      >Revenir à l’accueil de la box <span aria-hidden="true">↗</span></button
    >
  </section>
  <section class="panel">
    <div class="section-heading">
      <h2>Le dernier mot</h2>
      <span class="mini-label">CLASSEMENT FINAL</span>
    </div>
    <PlayerList {players} me={playerId} mode={game.mode} ranked />
  </section>
</div>
