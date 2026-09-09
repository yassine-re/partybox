<script lang="ts">
  import type { Player, ReactionChallenge, ReactionAssignment } from "$lib/api/types";
  import { validReactionAssignment } from "$lib/reaction";

  let {
    challenge,
    players,
    busy,
    onassign,
  }: {
    challenge: ReactionChallenge;
    players: Player[];
    busy: boolean;
    onassign: (assignment: ReactionAssignment) => void;
  } = $props();

  let soloPlayer = $state("");
  let soloButton = $state<"S2" | "S3">("S2");
  let s2Player = $state("");
  let s3Player = $state("");
  $effect(() => {
    if (!soloPlayer && players[0]) soloPlayer = players[0].id;
    if (!s2Player && players[0]) s2Player = players[0].id;
    if (!s3Player && players[1]) s3Player = players[1].id;
  });
  const assignment = $derived<ReactionAssignment>(challenge.kind === "solo"
    ? {
        button_s2_player_id: soloButton === "S2" ? soloPlayer || null : null,
        button_s3_player_id: soloButton === "S3" ? soloPlayer || null : null,
      }
    : {
        button_s2_player_id: s2Player || null,
        button_s3_player_id: s3Player || null,
      });
  const valid = $derived(validReactionAssignment(
    challenge,
    assignment.button_s2_player_id ?? "",
    assignment.button_s3_player_id ?? "",
  ));
</script>

<div class="reaction-modal" role="dialog" aria-modal="true" aria-labelledby="reaction-title">
  <div class="reaction-modal-card">
    <span class="eyebrow">{challenge.kind === "solo" ? "MODE SOLO" : "MODE DUEL"}</span>
    <h2 id="reaction-title">Qui relève le défi ?</h2>
    {#if challenge.kind === "solo"}
      <label for="reaction-player">JOUEUR</label>
      <select id="reaction-player" bind:value={soloPlayer}>
        {#each players as player}<option value={player.id}>{player.name}</option>{/each}
      </select>
      <label for="reaction-button">BOUTON PHYSIQUE</label>
      <select id="reaction-button" bind:value={soloButton}>
        <option value="S2">S2</option><option value="S3">S3</option>
      </select>
    {:else}
      <label for="reaction-s2">JOUEUR SUR S2</label>
      <select id="reaction-s2" bind:value={s2Player}>
        {#each players as player}<option value={player.id}>{player.name}</option>{/each}
      </select>
      <label for="reaction-s3">JOUEUR SUR S3</label>
      <select id="reaction-s3" bind:value={s3Player}>
        {#each players as player}<option value={player.id}>{player.name}</option>{/each}
      </select>
      {#if s2Player === s3Player}<p class="reaction-validation">Choisis deux joueurs différents.</p>{/if}
    {/if}
    <button class="button primary" disabled={busy || !valid} onclick={() => onassign(assignment)}>
      Armer la PartyBox <span>↗</span>
    </button>
  </div>
</div>
