<script lang="ts">
  import type { Box, GameMode } from "$lib/api/types";
  import { DEFAULT_GAME_MODE, GAME_MODES } from "$lib/game-modes";
  import ModeSelector from "./ModeSelector.svelte";
  let {
    box,
    nickname,
    busy,
    onsubmit,
  }: {
    box: Box;
    nickname: string;
    busy: boolean;
    onsubmit: (name: string, gameName: string, mode: GameMode) => void;
  } = $props();
  let name = $state("");
  let gameName = $state("La soirée du salon");
  let selectedMode = $state<GameMode>(DEFAULT_GAME_MODE);
  const mode = $derived(box.active_game?.mode ?? selectedMode);
  const presentation = $derived(GAME_MODES[mode]);
  $effect(() => {
    if (nickname && !name) name = nickname;
  });
</script>

<div class="entry-layout">
  <section class="entry-intro">
    <div class="eyebrow">
      <span class="live-dot" aria-hidden="true"></span> BOX {box.id} · PRÊTE À JOUER
    </div>
    <h1>{presentation.heading[0]}<br />{presentation.heading[1]}<br />{presentation.heading[2]} <em>{presentation.punchline}</em></h1>
    <p class="intro">
      {presentation.intro}
    </p>
    <div class="mode-ticket">
      <span class="ticket-star" aria-hidden="true">{presentation.symbol}</span>
      <div>
        <span class="eyebrow">LE MODE DE LA SOIRÉE</span>
        <h2>{presentation.label}</h2>
        <p>{presentation.minPlayers} joueurs et plus · Chacun pour soi</p>
      </div>
      <span class="ticket-number">{presentation.number}</span>
    </div>
    <p class="small-note">
      Reste sympa : chacun peut refuser de participer à une action.
    </p>
  </section>
  <section class="panel entry-panel">
    <span class="eyebrow"
      >{box.active_game ? "LA BANDE T’ATTEND" : "À TOI DE LANCER LE JEU"}</span
    >
    <h2>
      {box.active_game ? "Entre dans la partie." : "Ouvre les festivités."}
    </h2>
    <p class="muted">
      {box.active_game
        ? box.active_game.name
        : "Crée une partie, puis invite tes amis à scanner la box."}
    </p>
    {#if box.active_game}<div class="game-status">
        <span class="live-dot" aria-hidden="true"></span>{box.active_game
          .status === "playing"
          ? "Partie en cours · Tu peux encore rejoindre"
          : "Le lobby est ouvert"}
      </div>{/if}
    <form
      onsubmit={(event) => {
        event.preventDefault();
        onsubmit(name, gameName, mode);
      }}
    >
      {#if !box.active_game}
        <ModeSelector bind:value={selectedMode} disabled={busy} />
      {:else}
        <p class="small-note">{presentation.description}</p>
      {/if}
      <label for="nickname">TON PSEUDO</label>
      <input
        id="nickname"
        name="nickname"
        bind:value={name}
        maxlength="24"
        minlength="1"
        required
        placeholder="Ex. Agent Pingouin"
        autocomplete="nickname"
      />
      {#if !box.active_game}<label for="game-name">NOM DE LA PARTIE</label
        ><input
          id="game-name"
          name="game-name"
          bind:value={gameName}
          maxlength="60"
          minlength="1"
          required
        />{/if}
      <button
        class="button primary"
        type="submit"
        disabled={busy ||
          !name.trim() ||
          (!box.active_game && !gameName.trim())}
        >{busy
          ? "Un petit instant…"
          : box.active_game
            ? "Rejoindre la partie"
            : "Créer la partie"}<span aria-hidden="true">↗</span></button
      >
      <p class="form-hint">
        Pas de compte. Ton pseudo reste sur ce navigateur.
      </p>
    </form>
  </section>
</div>
