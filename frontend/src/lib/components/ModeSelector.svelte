<script lang="ts">
  import type { GameMode } from "$lib/api/types";
  import { DEFAULT_GAME_MODE, GAME_MODE_OPTIONS } from "$lib/game-modes";

  let { value = $bindable(DEFAULT_GAME_MODE), disabled = false }: {
    value?: GameMode;
    disabled?: boolean;
  } = $props();
</script>

<fieldset {disabled} class="mode-selector">
  <legend>CHOISIS TON MODE</legend>
  {#each GAME_MODE_OPTIONS as mode (mode.id)}
    <label class="mode-option" class:selected={value === mode.id}>
      <input type="radio" name="game-mode" value={mode.id} bind:group={value} />
      <span class="mode-symbol" aria-hidden="true">{mode.symbol}</span>
      <span class="mode-copy"><strong>{mode.label}</strong><small>{mode.description}</small></span>
      <span class="selection-mark" aria-hidden="true">{value === mode.id ? "✓" : "○"}</span>
    </label>
  {/each}
</fieldset>

<style>
  .mode-selector { border: 0; padding: 0; margin: 26px 0 0; min-width: 0; }
  legend { color: var(--muted); font: 600 10px monospace; letter-spacing: .13em; margin-bottom: 12px; }
  .mode-option { position: relative; display: flex; align-items: center; gap: 12px; margin: 0 0 10px; padding: 16px 14px; border: 1px solid var(--line); border-radius: 10px; cursor: pointer; letter-spacing: normal; }
  .mode-option.selected { border-color: var(--lime); background: #29331e; box-shadow: inset 3px 0 var(--lime); }
  input { position: absolute; width: 1px; height: 1px; min-height: 0; padding: 0; opacity: 0; }
  .mode-option:focus-within { outline: 3px solid var(--pink); outline-offset: 3px; }
  .mode-symbol { color: var(--lime); font-size: 30px; flex-shrink: 0; }
  .mode-copy { flex: 1; min-width: 0; font-family: "Avenir Next", "Trebuchet MS", sans-serif; }
  strong { display: block; font-size: 15px; color: var(--text); }
  small { display: block; color: var(--muted); font-size: 11px; line-height: 1.6; font-weight: 400; margin-top: 6px; }
  .selection-mark { color: var(--lime); font-size: 18px; }
  fieldset:disabled { opacity: .6; }
  fieldset:disabled .mode-option { cursor: wait; }
</style>
