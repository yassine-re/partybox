<script lang="ts">
  import type { Game, Player } from "$lib/api/types";
  import { GAME_MODES } from "$lib/game-modes";
  import PlayerList from "../PlayerList.svelte";

  let {
    game,
    players,
    playerId,
    busy,
    onshare,
    onstart,
  }: {
    game: Game;
    players: Player[];
    playerId: string;
    busy: boolean;
    onshare: () => void;
    onstart: () => void;
  } = $props();
  const me = $derived(players.find((player) => player.id === playerId));
  const mode = $derived(GAME_MODES[game.mode]);
</script>

<div class="game-grid">
  <section class="lobby-feature">
    <span class="eyebrow">TOUT COMMENCE ICI</span>
    <div class="lobby-star" aria-hidden="true">{mode.symbol}</div>
    <h2>Réunis<br />ta bande<span class="accent">.</span></h2>
    <p>{mode.lobbyDescription}</p>
    <button class="button outline" onclick={onshare}
      >Copier le lien d’invitation <span aria-hidden="true">↗</span></button
    >
    <p class="form-hint">Le même lien pour tous. Ou un simple scan NFC.</p>
  </section>
  <section class="panel">
    <div class="section-heading">
      <h2>Dans la place</h2>
      <span class="count-badge">{players.length}</span>
    </div>
    <PlayerList {players} me={playerId} />
    {#if me?.is_host}<button
        class="button primary"
        disabled={busy || players.length < mode.minPlayers}
        onclick={onstart}
        >{busy ? "Lancement…" : "Lancer la partie"}<span aria-hidden="true"
          >▶</span
        ></button
      >
      <p class="form-hint">
        {players.length < mode.minPlayers
          ? "Encore un ami et la soirée peut commencer."
          : mode.startHint}
      </p>{:else}<div class="waiting-notice">
        <span class="live-dot" aria-hidden="true"></span>En attendant que
        l’hôte lance la partie…
      </div>{/if}
  </section>
</div>
