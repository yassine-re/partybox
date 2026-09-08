<script lang="ts">
  import "../app.css";
  let { children } = $props();
</script>

<svelte:head>
  <title>PartyBox — La soirée passe en mode jeu</title>
  <meta
    name="description"
    content="Une box, tes amis, des missions secrètes. Le jeu de soirée qui se joue dans la vraie vie."
  />
</svelte:head>

<div class="site-shell">
  <header class="site-header">
    <a class="wordmark" href="/" aria-label="PartyBox, accueil"
      ><span class="brand-icon" aria-hidden="true">P<span>•</span></span
      >PARTY<span class="accent">BOX</span><span class="brand-dot">®</span></a
    >
    <span class="header-note"
      >LE JEU EST DANS LA PIÈCE<span class="live-dot" aria-hidden="true"
      ></span></span
    >
  </header>
  <main id="main">{@render children()}</main>
  <footer class="site-footer">
    <span>MOINS DE SCROLL. PLUS DE SOIRÉE.</span><span>PARTYBOX / V.01</span>
  </footer>
</div>
