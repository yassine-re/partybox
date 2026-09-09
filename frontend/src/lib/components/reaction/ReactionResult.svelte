<script lang="ts">
  import { onMount } from "svelte";
  import type { Player, ReactionChallenge } from "$lib/api/types";
  import { reactionResultText } from "$lib/reaction";

  let { challenge, players }: { challenge: ReactionChallenge; players: Player[] } = $props();
  let visible = $state(true);
  onMount(() => {
    const age = challenge.resolved_at ? Date.now() - new Date(challenge.resolved_at).getTime() : 0;
    const timer = setTimeout(() => (visible = false), Math.max(0, 9000 - age));
    return () => clearTimeout(timer);
  });
</script>

{#if visible}
  <section class="reaction-result" role="status" aria-live="assertive">
    <span aria-hidden="true">✦</span>
    <div><span class="eyebrow">RÉSULTAT RÉACTION</span><h2>{reactionResultText(challenge, players)}</h2></div>
  </section>
{/if}
