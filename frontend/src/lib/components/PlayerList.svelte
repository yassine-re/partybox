<script lang="ts">
  import type { GameMode, Player } from "$lib/api/types";
  import { DEFAULT_GAME_MODE, GAME_MODES } from "$lib/game-modes";
  let {
    players,
    me = "",
    ranked = false,
    mode = DEFAULT_GAME_MODE,
  }: { players: Player[]; me?: string; ranked?: boolean; mode?: GameMode } = $props();
</script>

<ol
  class="player-list"
  aria-label={ranked ? "Classement des joueurs" : "Joueurs dans le lobby"}
>
  {#each players as player, index (player.id)}
    <li class:own-player={player.id === me}>
      {#if ranked}<span class="rank" class:first={index === 0}
          >{index > 0 && players[index - 1].score === player.score
            ? players.findIndex((p) => p.score === player.score) + 1
            : index + 1}</span
        >{/if}
      <span
        class="avatar"
        class:avatar-pink={index % 3 === 1}
        class:avatar-blue={index % 3 === 2}
        >{player.name.slice(0, 1).toUpperCase()}</span
      >
      <div class="player-identity">
        <strong
          >{player.name}{#if player.id === me}<span class="you">
              TOI</span
            >{/if}</strong
        ><small
          >{ranked
            ? `${player.completed_missions} ${player.completed_missions > 1 ? GAME_MODES[mode].completedPlural : GAME_MODES[mode].completedSingular}`
            : player.is_host
              ? "Aux commandes de la soirée"
              : "Prêt à jouer"}</small
        >
      </div>
      {#if ranked}<span class="player-score"
          >{player.score}<small>PTS</small></span
        >{:else if player.is_host}<span class="host-tag">HÔTE</span>{:else}<span
          class="ready-check"
          aria-label="Prêt">✓</span
        >{/if}
    </li>
  {/each}
</ol>
