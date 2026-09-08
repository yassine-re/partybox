<script lang="ts">
  import { page } from "$app/state";
  import BoxExperience from "$lib/components/BoxExperience.svelte";
</script>

{#key page.params.boxId}<BoxExperience boxId={page.params.boxId ?? ""} />{/key}
