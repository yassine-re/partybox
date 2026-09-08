<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { api, ApiError } from "$lib/api";
  import type { Game } from "$lib/api/types";

  let {
    game,
    token,
    disabled = false,
    ongeneratingchange = () => {},
  }: {
    game: Game;
    token: string;
    disabled?: boolean;
    ongeneratingchange?: (generating: boolean) => void;
  } = $props();

  let vibe = $state<"chill" | "fun" | "chaos">("fun");
  let intensity = $state(7);
  let context = $state("");
  let generating = $state(false);
  let error = $state("");
  let available = $state(true);
  let customCount = $state(0);
  let remainingGenerations = $state<number | null>(null);
  let initialCheckDone = $state(false);

  async function checkStatus() {
    if (!token) return;
    try {
      const status = await api.aiMissionsStatus(game.id, token);
      available = status.available;
      customCount = status.count;
      remainingGenerations = status.remaining_generations;
    } catch {
      // Non-blocking: keep default available
    } finally {
      initialCheckDone = true;
    }
  }

  onMount(() => {
    void checkStatus();
  });

  onDestroy(() => {
    if (generating) ongeneratingchange(false);
  });

  async function handleGenerate() {
    if (
      generating ||
      disabled ||
      !available ||
      remainingGenerations === 0
    )
      return;
    generating = true;
    ongeneratingchange(true);
    error = "";
    try {
      const res = await api.aiGenerateMissions(game.id, token, {
        vibe,
        intensity,
        context: context.trim(),
        count: 20,
      });
      customCount = res.generated;
      await checkStatus();
    } catch (err) {
      if (err instanceof ApiError) {
        error = err.message;
      } else {
        error = "Impossible de joindre le générateur IA. Vérifie ta connexion.";
      }
      await checkStatus();
    } finally {
      generating = false;
      ongeneratingchange(false);
    }
  }
</script>

<section class="panel ai-panel" aria-label="Générateur de missions IA">
  <div class="ai-header">
    <div class="eyebrow">
      <span class="sparkle" aria-hidden="true">✨</span> MAÎTRE DU JEU IA
    </div>
    {#if customCount > 0}
      <span class="custom-badge">✓ {customCount} personnalisées</span>
    {/if}
  </div>

  {#if !available && initialCheckDone}
    <p class="ai-unavailable">
      La génération IA est indisponible (clé OpenAI non configurée ou mode non supporté). Les missions standards du jeu seront utilisées.
    </p>
  {:else}
    <div class="ai-form">
      <div class="field-group">
        <label class="field-label" for="vibe-buttons">Ambiance</label>
        <div id="vibe-buttons" class="vibe-pill-group" role="group" aria-label="Ambiance de la soirée">
          {#each ["chill", "fun", "chaos"] as option}
            <button
              type="button"
              class="vibe-pill"
              class:selected={vibe === option}
              disabled={generating || disabled}
              onclick={() => (vibe = option as "chill" | "fun" | "chaos")}
            >
              {option === "chill" ? "Chill" : option === "fun" ? "Fun" : "Chaos"}
            </button>
          {/each}
        </div>
      </div>

      <div class="field-group">
        <div class="slider-header">
          <label class="field-label" for="intensity-slider">Intensité</label>
          <span class="intensity-value">{intensity} / 10</span>
        </div>
        <input
          id="intensity-slider"
          type="range"
          min="1"
          max="10"
          step="1"
          bind:value={intensity}
          disabled={generating || disabled}
          class="intensity-slider"
        />
        <div class="slider-markers" aria-hidden="true">
          <span>1 (Tranquille)</span>
          <span>10 (Survolté)</span>
        </div>
      </div>

      <div class="field-group">
        <label class="field-label" for="ai-context">Contexte (optionnel)</label>
        <input
          id="ai-context"
          type="text"
          maxlength="300"
          placeholder="Ex. Soirée en appartement, pot de départ, week-end à la campagne…"
          bind:value={context}
          disabled={generating || disabled}
          class="context-input"
        />
        <span class="char-count">{context.length} / 300</span>
      </div>

      {#if error}
        <div class="message error" role="alert">
          {error}
        </div>
      {/if}

      {#if customCount > 0 && !generating}
        <div class="success-notice" role="status">
          <span class="check-icon" aria-hidden="true">✓</span>
          <span><strong>{customCount} missions personnalisées prêtes</strong> pour la partie.</span>
        </div>
      {/if}

      {#if remainingGenerations === 0}
        <div class="message error" role="status">
          La limite de générations IA pour cette partie est atteinte. Le catalogue actuel reste disponible.
        </div>
      {:else if remainingGenerations !== null}
        <p class="quota-hint">
          {remainingGenerations} génération{remainingGenerations > 1 ? "s" : ""} IA restante{remainingGenerations > 1 ? "s" : ""} pour cette partie.
        </p>
      {/if}

      <div class="ai-actions">
        <button
          type="button"
          class="button {customCount > 0 ? 'outline' : 'primary'}"
          disabled={generating || disabled || remainingGenerations === 0}
          onclick={handleGenerate}
        >
          {#if generating}
            <span class="spinner" aria-hidden="true">↻</span> L'IA prépare la partie…
          {:else if customCount > 0}
            Régénérer les missions <span aria-hidden="true">↻</span>
          {:else}
            Générer les missions <span aria-hidden="true">✨</span>
          {/if}
        </button>
      </div>

      <p class="form-hint">
        {#if customCount > 0}
          Les missions générées remplacent le catalogue standard pour cette partie.
        {:else}
          Personnalise le catalogue pour tes invités. Sans génération, les missions classiques s'appliquent.
        {/if}
      </p>
    </div>
  {/if}
</section>

<style>
  .ai-panel {
    margin-top: 20px;
    border-color: #3b4231;
    background: #181d16;
  }
  .ai-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }
  .sparkle {
    color: var(--lime);
    margin-right: 4px;
  }
  .custom-badge {
    background: #29331e;
    color: var(--lime);
    border: 1px solid var(--lime);
    border-radius: 999px;
    padding: 3px 10px;
    font-size: 11px;
    font-weight: 600;
  }
  .ai-unavailable {
    color: var(--muted);
    font-size: 13px;
    line-height: 1.5;
    margin: 0;
  }
  .ai-form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .field-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .field-label {
    color: var(--muted);
    font: 600 10px monospace;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .vibe-pill-group {
    display: flex;
    gap: 8px;
  }
  .vibe-pill {
    flex: 1;
    padding: 10px 12px;
    background: #20241e;
    color: var(--muted);
    border: 1px solid var(--line);
    border-radius: 8px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .vibe-pill:hover:not(:disabled) {
    color: var(--text);
    border-color: var(--muted);
  }
  .vibe-pill.selected {
    background: #29331e;
    color: var(--lime);
    border-color: var(--lime);
    box-shadow: inset 0 0 0 1px var(--lime);
  }
  .vibe-pill:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .slider-header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }
  .intensity-value {
    color: var(--lime);
    font-family: monospace;
    font-size: 13px;
    font-weight: 700;
  }
  .intensity-slider {
    width: 100%;
    accent-color: var(--lime);
    cursor: pointer;
    margin: 4px 0;
  }
  .slider-markers {
    display: flex;
    justify-content: space-between;
    font-size: 10px;
    color: var(--muted);
    font-family: monospace;
  }
  .context-input {
    width: 100%;
    box-sizing: border-box;
    padding: 10px 12px;
    background: #111410;
    color: var(--text);
    border: 1px solid var(--line);
    border-radius: 8px;
    font-size: 13px;
  }
  .context-input:focus {
    outline: none;
    border-color: var(--lime);
  }
  .char-count {
    align-self: flex-end;
    font-size: 10px;
    color: var(--muted);
    font-family: monospace;
  }
  .success-notice {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 12px;
    background: #232b1b;
    border: 1px solid var(--lime);
    border-radius: 8px;
    color: var(--lime);
    font-size: 13px;
  }
  .check-icon {
    font-weight: 700;
  }
  .ai-actions {
    display: flex;
    flex-direction: column;
    margin-top: 4px;
  }
  .quota-hint {
    margin: -4px 0 0;
    color: var(--muted);
    font-size: 11px;
    font-family: monospace;
  }
  .spinner {
    display: inline-block;
    animation: spin 1s linear infinite;
  }
  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
