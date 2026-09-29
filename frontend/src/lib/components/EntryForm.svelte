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
    onsubmit: (name: string, gameName: string, mode: GameMode, options: { durationMinutes: number; leaderboardVisibility: "visible" | "hidden"; validationMode: "trust" | "peer" }) => void;
  } = $props();
  let name = $state("");
  let gameName = $state("La soirée du salon");
  let selectedMode = $state<GameMode>(DEFAULT_GAME_MODE);
  let durationMinutes = $state(30);
  let leaderboardVisibility = $state<"visible" | "hidden">("visible");
  let validationMode = $state<"trust" | "peer">("trust");
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
        onsubmit(name, gameName, mode, { durationMinutes, leaderboardVisibility, validationMode });
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
      {#if !box.active_game}
        <details class="customize-game">
          <summary>Personnaliser la partie</summary>
          <fieldset class="duration-options" disabled={busy}>
            <legend>Durée de la partie</legend>
            {#each [{ value: 15, label: "15 min", hint: "Express" }, { value: 30, label: "30 min", hint: "Classique" }, { value: 60, label: "60 min", hint: "Longue" }, { value: 0, label: "∞", hint: "Toute la soirée" }] as option}
              <label class:selected={durationMinutes === option.value}>
                <input type="radio" name="duration" value={option.value} bind:group={durationMinutes} />
                <strong>{option.label}</strong><small>{option.hint}</small>
              </label>
            {/each}
          </fieldset>
          <fieldset class="settings-options" disabled={busy}>
            <legend>Classement</legend>
            <label><input type="radio" name="leaderboard" value="visible" bind:group={leaderboardVisibility} /> Visible pendant la partie</label>
            <label><input type="radio" name="leaderboard" value="hidden" bind:group={leaderboardVisibility} /> Caché jusqu’à la fin</label>
          </fieldset>
          <fieldset class="settings-options" disabled={busy}>
            <legend>Validation des missions</legend>
            <label><input type="radio" name="validation" value="trust" bind:group={validationMode} /> Mode confiance</label>
            <label><input type="radio" name="validation" value="peer" bind:group={validationMode} /> Validation par un autre joueur</label>
          </fieldset>
        </details>
      {/if}
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

<style>
  .customize-game { margin: 18px 0 22px; border: 1px solid var(--line); border-radius: 10px; }
  .customize-game summary { padding: 13px 14px; color: var(--muted); cursor: pointer; font-weight: 700; }
  .customize-game[open] summary { color: var(--lime); border-bottom: 1px solid var(--line); }
  fieldset { border: 0; padding: 14px; margin: 0; }
  legend { color: var(--muted); font-size: 12px; font-weight: 700; margin-bottom: 9px; }
  .duration-options { display: grid; grid-template-columns: repeat(4, 1fr); gap: 6px; }
  .duration-options label { display: grid; text-align: center; padding: 8px 3px; border: 1px solid var(--line); border-radius: 8px; cursor: pointer; }
  .duration-options label.selected { border-color: var(--lime); background: #29331e; }
  .duration-options input { position: absolute; opacity: 0; }
  .duration-options input:focus-visible + strong { outline: 2px solid var(--pink); }
  .duration-options small { color: var(--muted); font-size: 9px; }
  .settings-options { display: grid; gap: 8px; }
  .settings-options label { display: flex; align-items: center; gap: 8px; color: var(--text); font-size: 13px; }
  .settings-options input { accent-color: var(--lime); }
</style>
