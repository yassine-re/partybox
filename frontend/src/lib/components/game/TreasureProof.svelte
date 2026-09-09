<script lang="ts">
  import { onDestroy } from "svelte";
  import type { MissionProof, ProofResponse, ProofStatus } from "$lib/api/types";
  import { compressPhoto } from "$lib/photo";

  let { assignmentId, status, busy, onsubmit, oncomplete }: {
    assignmentId: string;
    status: ProofStatus;
    busy: boolean;
    onsubmit: (id: string, photo: Blob) => Promise<ProofResponse>;
    oncomplete: () => void;
  } = $props();
  let camera: HTMLInputElement;
  let gallery: HTMLInputElement;
  let preview = $state("");
  let photo = $state<Blob | null>(null);
  let preparing = $state(false);
  let sending = $state(false);
  let error = $state("");
  let proof = $state<MissionProof | null>(null);
  let disposed = false;
  const disabled = $derived(busy || preparing || sending);
  const exhausted = $derived(status.remaining_attempts === 0 && !status.accepted);

  function clearPhoto() {
    if (preview) URL.revokeObjectURL(preview);
    preview = "";
    photo = null;
  }

  onDestroy(() => { disposed = true; clearPhoto(); });

  async function select(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = "";
    if (!file || disabled || exhausted) return;
    preparing = true;
    error = "";
    proof = null;
    clearPhoto();
    try {
      const compressed = await compressPhoto(file);
      if (disposed) return;
      photo = compressed;
      preview = URL.createObjectURL(compressed);
    } catch (err) {
      if (!disposed) error = err instanceof Error ? err.message : "Photo illisible.";
    } finally {
      if (!disposed) preparing = false;
    }
  }

  async function send() {
    if (!photo || disabled || exhausted) return;
    sending = true;
    error = "";
    try {
      const response = await onsubmit(assignmentId, photo);
      if (disposed) return;
      proof = response.proof;
      clearPhoto();
    } catch (err) {
      if (!disposed) error = err instanceof Error ? err.message : "Analyse indisponible.";
    } finally {
      if (!disposed) sending = false;
    }
  }
</script>

<div class="treasure-proof" aria-busy={disabled}>
  <input class="file-input" bind:this={camera} type="file" accept="image/*" capture="environment" aria-label="Prendre une photo" onchange={select} disabled={disabled || exhausted} />
  <input class="file-input" bind:this={gallery} type="file" accept="image/*" aria-label="Choisir une photo dans la galerie" onchange={select} disabled={disabled || exhausted} />
  {#if preview}
    <img class="proof-preview" src={preview} alt="Aperçu de ta preuve avant envoi" />
  {/if}
  {#if preparing}<p role="status">Préparation de la photo…</p>{/if}
  {#if sending}<p role="status">Analyse de ta trouvaille… Cela peut prendre quelques secondes.</p>{/if}
  {#if proof}
    <div class="proof-verdict" role="status">
      <strong>{proof.verdict === "valid" ? "Trouvaille validée ✓" : proof.verdict === "invalid" ? "Pas encore 👀" : "Difficile à vérifier 🤔"}</strong>
      <p>{proof.reason}</p>
      {#if proof.verdict === "uncertain"}<p>Essaie une photo plus claire ou mieux cadrée.</p>{/if}
    </div>
  {/if}
  {#if error}<p role="alert">{error}</p>{/if}
  {#if status.accepted}
    <button class="button black" disabled={disabled} onclick={oncomplete}>Photo acceptée · terminer la validation ↗</button>
  {:else if exhausted}
    <p role="status">La limite de {status.max_attempts} analyses pour cette mission est atteinte.</p>
  {:else}
    {#if photo}<button class="button black" disabled={disabled} onclick={() => void send()}>{sending ? "Analyse…" : "Envoyer la photo pour validation ↗"}</button>{/if}
    <button class="button black" disabled={disabled} onclick={() => camera?.click()}>{proof || photo ? "Reprendre une photo" : "Prendre une photo"} ◎</button>
    <button class="text-button dark-text" disabled={disabled} onclick={() => gallery?.click()}>Choisir dans la galerie</button>
    <p class="proof-note">{status.remaining_attempts} analyse{status.remaining_attempts > 1 ? "s" : ""} restante{status.remaining_attempts > 1 ? "s" : ""} pour cette mission.</p>
  {/if}
  <p class="proof-note">Cadre ta trouvaille. La photo est envoyée à OpenAI pour vérification et n’est pas conservée par PartyBox. Évite les visages et les informations personnelles.</p>
</div>

<style>
  .treasure-proof { display: grid; gap: 12px; }
  .file-input { display: none; }
  .proof-preview { width: 100%; max-height: 300px; object-fit: contain; border-radius: 12px; background: #0001; }
  .proof-verdict { padding: 12px; border: 1px solid currentColor; border-radius: 12px; }
  p { margin: 0; line-height: 1.5; }
  .proof-note { font-size: 12px; opacity: .85; }
</style>
