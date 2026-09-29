<script lang="ts">
  import type { Game, Player } from "$lib/api/types";
  import { GAME_MODES } from "$lib/game-modes";
  import PlayerList from "../PlayerList.svelte";
  import AIMissionPanel from "./AIMissionPanel.svelte";

  let {
    game,
    players,
    playerId,
    token = "",
    busy,
    onshare,
    onstart,
    onaigeneratingchange = () => {},
  }: {
    game: Game;
    players: Player[];
    playerId: string;
    token?: string;
    busy: boolean;
    onshare: () => void;
    onstart: () => void;
    onaigeneratingchange?: (generating: boolean) => void;
  } = $props();
  let aiGenerating = $state(false);
  const me = $derived(players.find((player) => player.id === playerId));
  const mode = $derived(GAME_MODES[game.mode]);

  function handleAIGeneratingChange(generating: boolean) {
    aiGenerating = generating;
    onaigeneratingchange(generating);
  }
</script>

<div class="game-grid">
  <section class="lobby-feature">
    <span class="eyebrow">TOUT COMMENCE ICI</span>
    <div class="lobby-star" aria-hidden="true">{mode.symbol}</div>
    <h2>Réunis<br />ta bande<span class="accent">.</span></h2>
    <p>{mode.lobbyDescription}</p>
    <button class="button outline" onclick={onshare}
      >Inviter mes potes <span aria-hidden="true">↗</span></button
    >
    <p class="form-hint">Le même lien pour tous. Ou un simple scan NFC.</p>
  </section>
  <div class="lobby-sidebar">
    <section class="panel">
      <div class="section-heading">
        <h2>Dans la place</h2>
        <span class="count-badge">{players.length}</span>
      </div>
      <PlayerList {players} me={playerId} />
      {#if me?.is_host}<button
          class="button primary"
          disabled={busy || aiGenerating || players.length < mode.minPlayers}
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
    {#if me?.is_host && token && mode.supportsAIGeneration}
      <details class="customize-missions">
        <summary>✨ Personnaliser les missions</summary>
        <AIMissionPanel
          {game}
          {token}
          disabled={busy}
          ongeneratingchange={handleAIGeneratingChange}
        />
      </details>
    {/if}
  </div>
</div>

<style>
  .lobby-sidebar {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .customize-missions { border: 1px solid var(--line); border-radius: 12px; background: var(--surface, #171916); }
  .customize-missions summary { padding: 14px 16px; cursor: pointer; color: var(--muted); font-weight: 700; }
  .customize-missions[open] summary { color: var(--lime); border-bottom: 1px solid var(--line); }
  .customize-missions :global(.ai-mission-panel) { margin-top: 0; }
</style>
