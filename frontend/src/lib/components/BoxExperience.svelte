<script lang="ts">
  import { onMount } from "svelte";
  import { api, ApiError } from "$lib/api";
  import {
    clearSession,
    readNickname,
    readSession,
    saveSession,
  } from "$lib/api/session";
  import type {
    Box,
    Game,
    Mission,
    SavedSession,
    Session,
  } from "$lib/api/types";
  import EntryForm from "./EntryForm.svelte";
  import MissionCard from "./MissionCard.svelte";
  import PlayerList from "./PlayerList.svelte";

  let { boxId }: { boxId: string } = $props();
  let box = $state<Box | null>(null);
  let session = $state<SavedSession | null>(null);
  let playerId = $state("");
  let game = $state<Game | null>(null);
  let mission = $state<Mission | null>(null);
  let nickname = $state("");
  let loading = $state(true);
  let busy = $state(false);
  let error = $state("");
  let notice = $state("");
  let storageWarning = $state(false);
  let confirmEnd = $state(false);
  let tab = $state<"mission" | "leaderboard">("mission");
  let lastSync = $state("");
  let inFlight: Promise<void> | null = null;
  let disposed = false;
  const players = $derived(game?.players ?? []);
  const me = $derived(players.find((p) => p.id === playerId));
  const winners = $derived(
    players.filter((p) => p.score === players[0]?.score),
  );

  function showError(err: unknown) {
    error = err instanceof Error ? err.message : "Une erreur est survenue.";
  }

  async function fetchState() {
    const current = session;
    if (current) {
      if (!playerId) playerId = (await api.me(current.token)).id;
      const nextGame = await api.game(current.gameId, current.token);
      const nextMission =
        nextGame.status === "playing"
          ? (await api.mission(current.token)).mission
          : null;
      if (disposed) return;
      game = nextGame;
      mission = nextMission;
    } else {
      const nextBox = await api.box(boxId);
      if (disposed) return;
      box = nextBox;
    }
    lastSync = new Date().toLocaleTimeString("fr-FR", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  async function refresh() {
    if (inFlight) return inFlight;
    inFlight = fetchState();
    try {
      await inFlight;
    } finally {
      inFlight = null;
    }
  }

  async function sync() {
    try {
      await refresh();
      error = "";
    } catch (err) {
      if (
        err instanceof ApiError &&
        session &&
        [401, 404].includes(err.status)
      ) {
        clearSession(boxId);
        session = null;
        game = null;
        playerId = "";
        mission = null;
        notice =
          "Ta précédente session n’est plus disponible. Tu peux rejoindre à nouveau.";
        try {
          await refresh();
        } catch (next) {
          showError(next);
        }
      } else {
        showError(err);
      }
    }
  }

  onMount(() => {
    let timer: ReturnType<typeof setTimeout>;
    disposed = false;
    async function poll() {
      if (disposed) return;
      if (!busy && !document.hidden) await sync();
      if (!disposed) timer = setTimeout(poll, 3000);
    }
    async function initialize() {
      nickname = readNickname();
      const saved = readSession(boxId);
      if (saved) {
        session = saved;
        try {
          playerId = (await api.me(saved.token)).id;
        } catch (err) {
          if (err instanceof ApiError && err.status === 401) {
            clearSession(boxId);
            session = null;
          } else {
            showError(err);
          }
        }
      }
      await sync();
      loading = false;
      if (!disposed) timer = setTimeout(poll, 3000);
    }
    void initialize();
    const visible = () => {
      if (!document.hidden && !busy) void sync();
    };
    document.addEventListener("visibilitychange", visible);
    return () => {
      disposed = true;
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", visible);
    };
  });

  async function action(work: () => Promise<void>) {
    if (busy) return;
    busy = true;
    error = "";
    notice = "";
    try {
      if (inFlight) await inFlight.catch(() => undefined);
      await work();
      await refresh();
    } catch (err) {
      showError(err);
    } finally {
      busy = false;
    }
  }

  function acceptSession(result: Session) {
    session = { token: result.token, gameId: result.game.id };
    playerId = result.player.id;
    game = result.game;
    nickname = result.player.name;
    storageWarning = !saveSession(boxId, session, nickname);
  }

  function enter(name: string, gameName: string) {
    void action(async () => {
      const result = box?.active_game
        ? await api.join(box.active_game.id, name)
        : await api.create(boxId, gameName, name);
      acceptSession(result);
    });
  }

  function complete() {
    if (!session || !mission) return;
    const token = session.token,
      assignmentId = mission.id;
    void action(async () => {
      const result = await api.complete(token, assignmentId);
      mission = result.mission;
      notice = result.already_completed
        ? "Cette mission était déjà validée. À toi la suivante !"
        : `Mission accomplie ! +${result.awarded_points} points. Une nouvelle mission t’attend.`;
    });
  }

  function start() {
    if (session) {
      const current = session;
      void action(async () => {
        await api.start(current.gameId, current.token);
      });
    }
  }
  function end() {
    if (session) {
      const current = session;
      void action(async () => {
        await api.end(current.gameId, current.token);
        confirmEnd = false;
      });
    }
  }
  function backToBox() {
    void action(async () => {
      clearSession(boxId);
      session = null;
      game = null;
      mission = null;
      playerId = "";
      confirmEnd = false;
    });
  }
  async function share() {
    try {
      await navigator.clipboard.writeText(window.location.href);
      notice = "Lien copié. Envoie-le à tes amis !";
    } catch {
      notice = `Partage cette adresse : ${window.location.href}`;
    }
  }
</script>

<svelte:head
  ><title>{game ? game.name : `Box ${boxId}`} · PartyBox</title></svelte:head
>

{#if error}<div class="message error" role="alert">
    <span>{error}</span><button
      class="text-button"
      disabled={busy}
      onclick={() => void sync()}>Réessayer ↻</button
    >
  </div>{/if}
{#if notice}<div class="message success" role="status">{notice}</div>{/if}
{#if storageWarning}<div class="message error" role="alert">
    Le stockage du navigateur est désactivé. Garde cette page ouverte pour
    conserver ta session.
  </div>{/if}

{#if loading}
  <section class="loading-state" role="status">
    <span class="loading-star" aria-hidden="true">✳</span>
    <h1>On ouvre la box…</h1>
    <p class="muted">La soirée arrive.</p>
  </section>
{:else if !game && box}
  <EntryForm {box} {nickname} {busy} onsubmit={enter} />
{:else if game}
  <div class="game-heading">
    <div>
      <div class="eyebrow">
        <a href="/">ACCUEIL</a><span>/</span> BOX {boxId}<span>/</span> SECRET MISSIONS
      </div>
      <h1>{game.name}<span class="accent">.</span></h1>
    </div>
    <span class="status-pill"
      ><span class="live-dot" aria-hidden="true"></span>{game.status === "lobby"
        ? "LOBBY OUVERT"
        : game.status === "playing"
          ? "ÇA JOUE"
          : "PARTIE TERMINÉE"}</span
    >
  </div>

  {#if game.status === "lobby"}
    <div class="game-grid">
      <section class="lobby-feature">
        <span class="eyebrow">TOUT COMMENCE ICI</span>
        <div class="lobby-star" aria-hidden="true">✳</div>
        <h2>Réunis<br />ta bande<span class="accent">.</span></h2>
        <p>
          Tout le monde est là ? Une fois la partie lancée, chacun reçoit sa
          mission. Garde-la pour toi.
        </p>
        <button class="button outline" onclick={share}
          >Copier le lien d’invitation <span aria-hidden="true">↗</span></button
        >
        <p class="form-hint">Le même lien pour tous. Ou un simple scan NFC.</p>
      </section>
      <section class="panel">
        <div class="section-heading">
          <h2>Dans la place</h2>
          <span class="count-badge">{players.length}</span>
        </div>
        <PlayerList {players} me={playerId} />
        {#if me?.is_host}<button
            class="button primary"
            disabled={busy || players.length < 2}
            onclick={start}
            >{busy ? "Lancement…" : "Lancer la partie"}<span aria-hidden="true"
              >▶</span
            ></button
          >
          <p class="form-hint">
            {players.length < 2
              ? "Encore un ami et la soirée peut commencer."
              : "Ta mission sera attribuée au lancement."}
          </p>{:else}<div class="waiting-notice">
            <span class="live-dot" aria-hidden="true"></span>En attendant que
            l’hôte lance la partie…
          </div>{/if}
      </section>
    </div>
  {:else if game.status === "playing"}
    <div class="mobile-tabs" aria-label="Affichage du jeu">
      <button
        class:active={tab === "mission"}
        aria-pressed={tab === "mission"}
        onclick={() => (tab = "mission")}>Ma mission</button
      ><button
        class:active={tab === "leaderboard"}
        aria-pressed={tab === "leaderboard"}
        onclick={() => (tab = "leaderboard")}>Classement</button
      >
    </div>
    <div class="game-grid play-grid">
      <div class:mobile-hidden={tab !== "mission"}>
        <div class="personal-stats">
          <span>SALUT, <strong>{me?.name ?? "AGENT"}</strong></span><span
            ><strong>{me?.score ?? 0}</strong> PTS
            <span class="stat-divider">/</span>
            {me?.completed_missions ?? 0} MISSION(S)</span
          >
        </div>
        {#if mission}<MissionCard
            {mission}
            {busy}
            oncomplete={complete}
          />{:else}<section class="panel">
            <h2>Ta mission arrive…</h2>
            <button class="button outline" onclick={() => void sync()}
              >Actualiser</button
            >
          </section>{/if}
      </div>
      <section
        class="panel leaderboard-panel"
        class:mobile-hidden={tab !== "leaderboard"}
      >
        <div class="section-heading">
          <h2>Le classement</h2>
          <span class="mini-label">EN DIRECT*</span>
        </div>
        <PlayerList {players} me={playerId} ranked />
        <p class="form-hint">* Actualisé toutes les 3 secondes.</p>
      </section>
    </div>
  {:else}
    <div class="final-layout">
      <section class="final-feature">
        <div class="eyebrow">LES SECRETS SONT DÉVOILÉS</div>
        <span class="final-trophy" aria-hidden="true">✳</span>
        <h2>
          Bien joué,<br /><em>{winners.map((p) => p.name).join(" & ")}.</em>
        </h2>
        <p>
          {winners.length > 1
            ? "La victoire se partage ce soir."
            : "La soirée a son champion."}
          {players.reduce((sum, p) => sum + p.completed_missions, 0)} missions accomplies
          ensemble.
        </p>
        <div class="final-score">
          {players[0]?.score ?? 0}<span>POINTS AU SOMMET</span>
        </div>
        <button class="button primary" onclick={backToBox} disabled={busy}
          >Revenir à l’accueil de la box <span aria-hidden="true">↗</span
          ></button
        >
      </section>
      <section class="panel">
        <div class="section-heading">
          <h2>Le dernier mot</h2>
          <span class="mini-label">CLASSEMENT FINAL</span>
        </div>
        <PlayerList {players} me={playerId} ranked />
      </section>
    </div>
  {/if}

  <div class="game-bottom">
    <span class="sync-label"
      >↻ {lastSync ? `Dernière synchro ${lastSync}` : "Synchronisation…"}</span
    ><button class="text-button" disabled={busy} onclick={() => void sync()}
      >Actualiser</button
    >
    {#if me?.is_host && game.status !== "ended"}<button
        class="text-button end-button"
        disabled={busy}
        onclick={() => (confirmEnd = !confirmEnd)}
        >{game.status === "lobby"
          ? "Fermer le lobby"
          : "Terminer la partie"}</button
      >{/if}
  </div>
  {#if confirmEnd && game.status !== "ended"}<section
      class="end-confirm"
      aria-label="Confirmation de fin"
    >
      <div>
        <strong>On s’arrête là ?</strong>
        <p>Les scores seront figés pour tout le monde.</p>
      </div>
      <button
        class="button outline"
        disabled={busy}
        onclick={() => (confirmEnd = false)}>Continuer à jouer</button
      ><button class="button danger" disabled={busy} onclick={end}
        >Oui, terminer</button
      >
    </section>{/if}
{:else}
  <section class="loading-state">
    <h1>Impossible d’ouvrir la box<span class="accent">.</span></h1>
    <p class="muted">
      Vérifie le code de ta box ou réessaie si la connexion est interrompue.
    </p>
    <a class="button primary" href="/">Revenir à l’accueil ↗</a>
  </section>
{/if}
