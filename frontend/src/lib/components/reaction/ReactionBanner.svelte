<script lang="ts">
  import type { ReactionState } from "$lib/api/types";

  let { state }: { state: ReactionState } = $props();
  const active = $derived(state.challenge && ["awaiting_assignment", "awaiting_device", "armed"].includes(state.challenge.status));
</script>

{#if state.enabled && !state.device_online}
  <div class="reaction-offline" role="status">○ PartyBox physique hors ligne — le jeu web continue normalement.</div>
{/if}
{#if active}
  <section class="reaction-banner" aria-live="polite">
    <span class="reaction-pulse" aria-hidden="true">⚡</span>
    <div>
      <span class="eyebrow">INTERRUPTION SURPRISE</span>
      <h2>Défi réaction !</h2>
      <p>{state.challenge?.status === "awaiting_assignment"
          ? "L’hôte choisit qui prend S2 et S3."
          : state.challenge?.status === "awaiting_device"
            ? "Commande envoyée à la PartyBox…"
            : "Préparez-vous près de la PartyBox. Attendez la LED verte !"}</p>
    </div>
  </section>
{/if}
