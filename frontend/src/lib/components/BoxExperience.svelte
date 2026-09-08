<script lang="ts">
  import { onMount } from "svelte";
  import { api, ApiError } from "$lib/api";
  import { DEFAULT_GAME_MODE, GAME_MODES } from "$lib/game-modes";
  import {
    clearSession,
    readNickname,
    readSession,
    saveSession,
  } from "$lib/api/session";
  import { GameRealtime, type RealtimeStatus } from "$lib/realtime";
  import type {
    Box,
    Game,
    GameMode,
    Mission,
    SavedSession,
    Session,
  } from "$lib/api/types";
  import EntryForm from "./EntryForm.svelte";
  import FinalResults from "./game/FinalResults.svelte";
  import GameHeader from "./game/GameHeader.svelte";
  import LobbyView from "./game/LobbyView.svelte";
  import PlayingView from "./game/PlayingView.svelte";

  let { boxId }: { boxId: string } = $props();
  let box = $state<Box | null>(null);
  let session = $state<SavedSession | null>(null);
  let playerId = $state("");
  let game = $state<Game | null>(null);
  let mission = $state<Mission | null>(null);
  let nickname = $state("");
  let loading = $state(true);
  let busy = $state(false);
  let aiGenerating = $state(false);
  let error = $state("");
  let notice = $state("");
  let storageWarning = $state(false);
  let confirmEnd = $state(false);
  let tab = $state<"mission" | "leaderboard">("mission");
  let lastSync = $state("");
  let realtimeStatus = $state<RealtimeStatus>("offline");
  let inFlight: Promise<void> | null = null;
  let realtimeClient: GameRealtime | null = null;
  let realtimeRefreshTimer: ReturnType<typeof setTimeout> | null = null;
  let realtimeRefreshPending = false;
  let disposed = false;
  const players = $derived(game?.players ?? []);
  const mode = $derived(GAME_MODES[game?.mode ?? DEFAULT_GAME_MODE]);
  const me = $derived(players.find((p) => p.id === playerId));

  function showError(err: unknown) {
    error = err instanceof Error ? err.message : "Une erreur est survenue.";
  }

  function stopRealtime() {
    realtimeClient?.stop();
    realtimeClient = null;
    realtimeStatus = "offline";
  }

  function queueRealtimeRefresh() {
    realtimeRefreshPending = true;
    if (realtimeRefreshTimer) clearTimeout(realtimeRefreshTimer);
    realtimeRefreshTimer = setTimeout(() => {
      realtimeRefreshTimer = null;
      if (disposed) return;
      if (busy || inFlight) return;
      realtimeRefreshPending = false;
      void sync();
    }, 120);
  }

  function startRealtime() {
    stopRealtime();
    const current = session;
    if (!current || disposed) return;
    const client = new GameRealtime(current.gameId, current.token, {
      onEvent: () => queueRealtimeRefresh(),
      onStatus: (status) => {
        const wasConnected = realtimeStatus === "connected";
        realtimeStatus = status;
        // Catch up on anything committed while a phone was asleep or the
        // connection was being established.
        if (status === "connected" && !wasConnected) queueRealtimeRefresh();
      },
    });
    realtimeClient = client;
    client.start();
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
      if (realtimeRefreshPending && !busy && !disposed) {
        queueRealtimeRefresh();
      }
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
        stopRealtime();
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
    let fallbackTimer: ReturnType<typeof setInterval> | undefined;
    disposed = false;
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
      if (session) startRealtime();
      // No polling while realtime is healthy. This only covers a broken
      // socket (and lets a pre-game entry screen discover a new lobby).
      fallbackTimer = setInterval(() => {
        if (
          !disposed &&
          !busy &&
          !document.hidden &&
          realtimeStatus !== "connected"
        ) {
          void sync();
        }
      }, 25_000);
    }
    void initialize();
    const visible = () => {
      if (!document.hidden) {
        realtimeClient?.reconnectNow();
        if (!busy) void sync();
      }
    };
    document.addEventListener("visibilitychange", visible);
    return () => {
      disposed = true;
      stopRealtime();
      if (fallbackTimer) clearInterval(fallbackTimer);
      if (realtimeRefreshTimer) clearTimeout(realtimeRefreshTimer);
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
      if (realtimeRefreshPending) queueRealtimeRefresh();
    }
  }

  function acceptSession(result: Session) {
    session = { token: result.token, gameId: result.game.id };
    playerId = result.player.id;
    game = result.game;
    nickname = result.player.name;
    storageWarning = !saveSession(boxId, session, nickname);
    startRealtime();
  }

  function enter(name: string, gameName: string, selectedMode: GameMode) {
    void action(async () => {
      const result = box?.active_game
        ? await api.join(box.active_game.id, name)
        : await api.create(boxId, gameName, name, selectedMode);
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
        ? mode.alreadyCompletedMessage
        : mode.completionMessage(result.awarded_points);
    });
  }

  function start() {
    if (session && !aiGenerating) {
      const current = session;
      void action(async () => {
        await api.start(current.gameId, current.token);
      });
    }
  }
  function end() {
    if (session && !aiGenerating) {
      const current = session;
      void action(async () => {
        await api.end(current.gameId, current.token);
        confirmEnd = false;
      });
    }
  }
  function backToBox() {
    void action(async () => {
      stopRealtime();
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
  <GameHeader {boxId} {game} />

  {#if game.status === "lobby"}
    <LobbyView
      {game}
      {players}
      {playerId}
      token={session?.token}
      {busy}
      onshare={share}
      onstart={start}
      onaigeneratingchange={(generating) => (aiGenerating = generating)}
    />
  {:else if game.status === "playing"}
    <PlayingView
      {game}
      {players}
      {playerId}
      {mission}
      {busy}
      {realtimeStatus}
      {tab}
      oncomplete={complete}
      onrefresh={() => void sync()}
      ontabchange={(nextTab) => (tab = nextTab)}
    />
  {:else}
    <FinalResults {game} {players} {playerId} {busy} onback={backToBox} />
  {/if}

  <div class="game-bottom">
    <span class="sync-label"
      >{realtimeStatus === "connected"
        ? "● TEMPS RÉEL"
        : realtimeStatus === "offline"
          ? "○ HORS LIGNE"
          : "○ RECONNEXION"} · {lastSync
        ? `Dernière synchro ${lastSync}`
        : "Synchronisation…"}</span
    ><button class="text-button" disabled={busy} onclick={() => void sync()}
      >Actualiser</button
    >
    {#if me?.is_host && game.status !== "ended"}<button
        class="text-button end-button"
        disabled={busy || aiGenerating}
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
        disabled={busy || aiGenerating}
        onclick={() => (confirmEnd = false)}>Continuer à jouer</button
      ><button class="button danger" disabled={busy || aiGenerating} onclick={end}
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
