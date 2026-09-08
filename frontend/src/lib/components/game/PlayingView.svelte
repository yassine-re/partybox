<script lang="ts">
  import type { Game, Mission, Player } from "$lib/api/types";
  import type { RealtimeStatus } from "$lib/realtime";
  import { GAME_MODES } from "$lib/game-modes";
  import MissionCard from "../MissionCard.svelte";
  import PlayerList from "../PlayerList.svelte";

  type GameTab = "mission" | "leaderboard";

  let {
    game,
    players,
    playerId,
    mission,
    busy,
    realtimeStatus,
    tab,
    oncomplete,
    onrefresh,
    ontabchange,
  }: {
    game: Game;
    players: Player[];
    playerId: string;
    mission: Mission | null;
    busy: boolean;
    realtimeStatus: RealtimeStatus;
    tab: GameTab;
    oncomplete: () => void;
    onrefresh: () => void;
    ontabchange: (tab: GameTab) => void;
  } = $props();
  const me = $derived(players.find((player) => player.id === playerId));
  const mode = $derived(GAME_MODES[game.mode]);
</script>

<div class="mobile-tabs" aria-label="Affichage du jeu">
  <button
    class:active={tab === "mission"}
    aria-pressed={tab === "mission"}
    onclick={() => ontabchange("mission")}>{mode.tabLabel}</button
  ><button
    class:active={tab === "leaderboard"}
    aria-pressed={tab === "leaderboard"}
    onclick={() => ontabchange("leaderboard")}>Classement</button
  >
</div>
<div class="game-grid play-grid">
  <div class:mobile-hidden={tab !== "mission"}>
    <div class="personal-stats">
      <span>SALUT, <strong>{me?.name ?? mode.playerLabel}</strong></span><span
        ><strong>{me?.score ?? 0}</strong> PTS
        <span class="stat-divider">/</span>
        {me?.completed_missions ?? 0} {mode.statsLabel}</span
      >
    </div>
    {#if mission}<MissionCard
        {mission}
        mode={game.mode}
        {busy}
        oncomplete={oncomplete}
      />{:else}<section class="panel">
        <h2>{mode.loadingText}</h2>
        <button class="button outline" onclick={onrefresh}>Actualiser</button>
      </section>{/if}
  </div>
  <section
    class="panel leaderboard-panel"
    class:mobile-hidden={tab !== "leaderboard"}
  >
    <div class="section-heading">
      <h2>Le classement</h2>
      <span class="mini-label realtime-label">
        <span
          class:reconnecting={realtimeStatus !== "connected"}
          class="live-dot"
          aria-hidden="true"
        ></span>{realtimeStatus === "connected"
          ? "EN DIRECT"
          : "RECONNEXION…"}
      </span>
    </div>
    <PlayerList {players} mode={game.mode} me={playerId} ranked />
    <p class="form-hint">
      {realtimeStatus === "connected"
        ? "Classement synchronisé en temps réel."
        : "Actualisation de secours pendant la reconnexion."}
    </p>
  </section>
</div>
