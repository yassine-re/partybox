<script lang="ts">
  import { goto } from "$app/navigation";
  let boxId = $state("PB001");
</script>

<section class="home-hero">
  <div class="eyebrow">
    <span class="live-dot" aria-hidden="true"></span> PARTY GAME · DANS LA VRAIE VIE
  </div>
  <h1>La soirée.<br />En mode <em>jeu.</em></h1>
  <p class="intro">
    Tes amis n’ont aucune idée de ce que tu prépares.<br class="desktop-only" /> Et
    c’est exactement le principe.
  </p>
  <form
    class="home-form"
    onsubmit={(event) => {
      event.preventDefault();
      void goto(`/box/${encodeURIComponent(boxId.trim())}`);
    }}
  >
    <label for="box-id">LE CODE DE TA BOX</label>
    <div class="input-row">
      <input
        id="box-id"
        name="box"
        bind:value={boxId}
        pattern={"[A-Za-z0-9_\\-]{1,40}"}
        maxlength="40"
        required
        autocomplete="off"
      /><button class="button primary" type="submit"
        >C’est parti <span aria-hidden="true">↗</span></button
      >
    </div>
    <p class="form-hint">Ou approche ton téléphone du tag NFC de la box.</p>
  </form>
  <div class="home-art" aria-hidden="true">
    <div class="orbit orbit-one"></div>
    <div class="orbit orbit-two"></div>
    <div class="physical-box">
      <span>PARTY<br />BOX<span class="box-star">✳</span></span><small
        >TAP TO PLAY ↗</small
      >
    </div>
    <span class="art-label">100 % IRL<br />0 % SMALL TALK</span>
  </div>
</section>
<section class="how-it-works" aria-label="Comment jouer">
  <div>
    <span class="step-number">01 /</span>
    <h2>Entre dans la partie.</h2>
    <p>Un pseudo et tes amis. C’est tout ce qu’il te faut.</p>
  </div>
  <div>
    <span class="step-number">02 /</span>
    <h2>Joue l’air de rien.</h2>
    <p>Une mission secrète à accomplir dans la soirée.</p>
  </div>
  <div>
    <span class="step-number">03 /</span>
    <h2>Marque le coup.</h2>
    <p>Valide, gagne des points et remonte au classement.</p>
  </div>
</section>
