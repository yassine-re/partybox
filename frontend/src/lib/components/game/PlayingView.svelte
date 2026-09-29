<script lang="ts">
  import type {
    ChaosState,
    Game,
    Mission,
    MissionFeedbackRating,
    Player,
    ProofResponse,
    ProofStatus,
    ReactionAssignment,
    ReactionState,
  } from "$lib/api/types";
  import type { RealtimeStatus } from "$lib/realtime";
  import { GAME_MODES } from "$lib/game-modes";
  import MissionCard from "../MissionCard.svelte";
  import PlayerList from "../PlayerList.svelte";
  import ChaosBanner from "./ChaosBanner.svelte";
  import TreasureProof from "./TreasureProof.svelte";
  import MissionFeedback from "./MissionFeedback.svelte";
  import ReactionBanner from "../reaction/ReactionBanner.svelte";
  import ReactionAssignmentModal from "../reaction/ReactionAssignmentModal.svelte";
  import ReactionPlayerOverlay from "../reaction/ReactionPlayerOverlay.svelte";
  import ReactionResult from "../reaction/ReactionResult.svelte";
  import { canAssignReaction } from "$lib/reaction";
  import { onMount } from "svelte";

  type GameTab = "mission" | "leaderboard";

  let {
    game,
    players,
    playerId,
    mission,
    busy,
    realtimeStatus,
    tab,
    oncomplete,
    onskipmission = () => {},
    onrefresh,
    ontabchange,
    feedbackAssignmentId = null,
    feedbackBusy = false,
    feedbackError = "",
    onfeedback,
    onfeedbackskip,
    chaosState = null,
    proofStatus = null,
    onproofsubmit,
    reactionState = null,
    onreactionassign,
    validationRequests = [],
    validationPending = false,
    onvalidationresolve = () => {},
  }: {
    game: Game;
    players: Player[];
    playerId: string;
    mission: Mission | null;
    busy: boolean;
    realtimeStatus: RealtimeStatus;
    tab: GameTab;
    oncomplete: () => void;
    onskipmission?: () => void;
    onrefresh: () => void;
    ontabchange: (tab: GameTab) => void;
    feedbackAssignmentId?: string | null;
    feedbackBusy?: boolean;
    feedbackError?: string;
    onfeedback: (rating: MissionFeedbackRating) => void;
    onfeedbackskip: () => void;
    chaosState?: ChaosState | null;
    proofStatus?: ProofStatus | null;
    onproofsubmit: (id: string, image: Blob) => Promise<ProofResponse>;
    reactionState?: ReactionState | null;
    onreactionassign: (challengeId: string, assignment: ReactionAssignment) => void;
    validationRequests?: import("$lib/api/types").ValidationRequest[];
    validationPending?: boolean;
    onvalidationresolve?: (requestId: string, approved: boolean) => void;
  } = $props();
  const me = $derived(players.find((player) => player.id === playerId));
  const mode = $derived(GAME_MODES[game.mode]);
  const reactionChallenge = $derived(reactionState?.challenge ?? null);
  const missionBusy = $derived(busy || validationPending);
  let now = $state(Date.now());
  let previousPhase = $state("");
  let lastChaosSequence = $state<number | null>(null);
  const remainingSeconds = $derived(game.ends_at ? Math.max(0, Math.ceil((new Date(game.ends_at).getTime() - now) / 1000)) : null);
  const phase = $derived(remainingSeconds !== null && remainingSeconds <= 60 ? "final" : remainingSeconds !== null && remainingSeconds <= 300 ? "five" : "normal");
  const timerLabel = $derived(remainingSeconds === null ? "" : `${Math.floor(remainingSeconds / 60).toString().padStart(2, "0")}:${(remainingSeconds % 60).toString().padStart(2, "0")}`);
  onMount(() => {
    if (!game.ends_at) return;
    const timer = setInterval(() => { now = Date.now(); }, 1000);
    return () => clearInterval(timer);
  });
  $effect(() => {
    if (phase !== previousPhase && phase === "final" && typeof navigator !== "undefined" && navigator.vibrate) navigator.vibrate([120, 60, 120]);
    previousPhase = phase;
  });
  $effect(() => {
    const sequence = chaosState?.sequence;
    if (sequence === undefined || sequence === null) return;
    if (lastChaosSequence !== null && sequence !== lastChaosSequence && typeof navigator !== "undefined" && navigator.vibrate) navigator.vibrate([45, 25, 45]);
    lastChaosSequence = sequence;
  });
</script>

{#if game.ends_at}
  <div class="game-timer" class:urgent={phase === "final"} aria-live="off">
    <span aria-hidden="true">{phase === "final" ? "🔥" : "⏳"}</span>
    <strong>{timerLabel}</strong><span>restantes</span>
  </div>
{/if}
{#if phase === "five"}
  <aside class="phase-notice" role="status">⏳ Plus que 5 minutes <span>C’est le moment de terminer ta mission.</span></aside>
{:else if phase === "final"}
  <aside class="last-minute" role="status"><span>🔥 DERNIÈRE MINUTE</span><strong>Toutes les missions valent x2.</strong></aside>
{/if}
{#if validationRequests.length}
  <section class="validation-inbox" aria-live="polite">
    <h2>À confirmer</h2>
    {#each validationRequests as request (request.id)}
      <article><p><strong>{request.player_name}</strong> a terminé :</p><blockquote>« {request.mission_text} »</blockquote>
        <div><button class="button primary" onclick={() => onvalidationresolve(request.id, true)}>Confirmer</button><button class="button outline" onclick={() => onvalidationresolve(request.id, false)}>Refuser</button></div>
      </article>
    {/each}
  </section>
{/if}

<div class="mobile-tabs" aria-label="Affichage du jeu">
  <button
    class:active={tab === "mission"}
    aria-pressed={tab === "mission"}
    onclick={() => ontabchange("mission")}>{mode.tabLabel}</button
  >{#if game.leaderboard_visibility !== "hidden"}<button
    class:active={tab === "leaderboard"}
    aria-pressed={tab === "leaderboard"}
    onclick={() => ontabchange("leaderboard")}>Classement</button
  >{/if}
</div>

<style>
  .game-timer { display: flex; align-items: center; justify-content: center; gap: 8px; margin: 0 auto 16px; padding: 10px 16px; width: fit-content; border: 1px solid var(--line); border-radius: 999px; color: var(--muted); font-size: 13px; }
  .game-timer strong { color: var(--text); font: 800 20px monospace; }
  .game-timer.urgent { color: #ff9cab; border-color: var(--pink); animation: timer-pulse 1s ease-in-out infinite alternate; }
  .game-timer.urgent strong { color: var(--pink); }
  .phase-notice, .last-minute { margin: 0 auto 18px; max-width: 720px; padding: 13px 16px; border-radius: 10px; text-align: center; background: #272a20; border: 1px solid #737c40; font-weight: 800; }
  .phase-notice span { display: block; margin-top: 3px; color: var(--muted); font-size: 13px; font-weight: 500; }
  .last-minute { display: grid; gap: 4px; color: #fff; border-color: var(--pink); background: linear-gradient(110deg, #642e3b, #332234); animation: final-pulse 1s ease-in-out infinite alternate; }
  .last-minute span { color: #ffc1cb; font: 900 13px monospace; letter-spacing: .12em; }
  .last-minute strong { font-size: 20px; }
  .validation-inbox { margin-bottom: 18px; padding: 16px; border: 1px solid var(--lime); border-radius: 12px; background: #1e2418; }
  .validation-inbox h2 { margin: 0 0 10px; }
  .validation-inbox article { padding: 12px 0; border-top: 1px solid var(--line); }
  .validation-inbox p, .validation-inbox blockquote { margin: 0 0 8px; }
  .validation-inbox blockquote { color: var(--lime); }
  .validation-inbox article div { display: flex; gap: 8px; }
  .chaos-impact { animation: chaos-arrival .55s cubic-bezier(.2,.8,.2,1); }
  @keyframes timer-pulse { to { box-shadow: 0 0 20px #ff6b803c; transform: scale(1.03); } }
  @keyframes final-pulse { to { box-shadow: 0 0 30px #ff66803d; transform: scale(1.01); } }
  @keyframes chaos-arrival { 0% { transform: scale(.96) translateY(-10px); filter: brightness(1.8); } 60% { transform: scale(1.015); } 100% { transform: scale(1); filter: brightness(1); } }
  @media (prefers-reduced-motion: reduce) { .game-timer, .last-minute, .chaos-impact { animation: none; } }
</style>
{#if game.mode === "chaos" && chaosState}
  {#key chaosState.sequence}<div class="chaos-impact"><ChaosBanner state={chaosState} /></div>{/key}
{/if}
{#if reactionState}<ReactionBanner state={reactionState} />{/if}
{#if reactionChallenge?.status === "resolved"}
  <ReactionResult challenge={reactionChallenge} {players} />
{/if}
{#if reactionChallenge && canAssignReaction(reactionState, Boolean(me?.is_host))}
  <ReactionAssignmentModal
    challenge={reactionChallenge}
    {players}
    {busy}
    onassign={(assignment) => onreactionassign(reactionChallenge.id, assignment)}
  />
{/if}
{#if reactionChallenge}
  <ReactionPlayerOverlay challenge={reactionChallenge} {playerId} />
{/if}
<div class="game-grid play-grid">
  <div class:mobile-hidden={tab !== "mission"}>
    <div class="personal-stats">
      <span>SALUT, <strong>{me?.name ?? mode.playerLabel}</strong></span><span
        ><strong>{me?.score ?? 0}</strong> PTS
        <span class="stat-divider">/</span>
        {me?.completed_missions ?? 0} {mode.statsLabel}</span
      >
    </div>
    {#if mission}<MissionCard
        {mission}
        mode={game.mode}
        busy={missionBusy}
        validationPending={validationPending}
        oncomplete={oncomplete}
        onskip={game.mode === "treasure_hunt" ? onskipmission : undefined}
        actions={game.mode === "treasure_hunt" && proofStatus?.available ? photoActions : undefined}
      />{:else}<section class="panel">
        <h2>{mode.loadingText}</h2>
        <button class="button outline" onclick={onrefresh}>Actualiser</button>
      </section>{/if}
    {#if feedbackAssignmentId}
      <MissionFeedback
        assignmentId={feedbackAssignmentId}
        busy={feedbackBusy}
        error={feedbackError}
        onsubmit={onfeedback}
        onskip={onfeedbackskip}
      />
    {/if}
  </div>
  {#if game.leaderboard_visibility !== "hidden"}<section
    class="panel leaderboard-panel"
    class:mobile-hidden={tab !== "leaderboard"}
  >
    <div class="section-heading">
      <h2>Le classement</h2>
      <span class="mini-label realtime-label">
        <span
          class:reconnecting={realtimeStatus !== "connected"}
          class="live-dot"
          aria-hidden="true"
        ></span>{realtimeStatus === "connected"
          ? "EN DIRECT"
          : "RECONNEXION…"}
      </span>
    </div>
    <PlayerList {players} mode={game.mode} me={playerId} ranked />
    <p class="form-hint">
      {realtimeStatus === "connected"
        ? "Classement synchronisé en temps réel."
        : "Actualisation de secours pendant la reconnexion."}
    </p>
  </section>{:else}<section class="panel leaderboard-panel" class:mobile-hidden={tab !== "leaderboard"}>
      <div class="section-heading"><h2>Classement</h2><span class="mini-label">CACHÉ</span></div>
      <p>Les scores seront révélés à la fin de la partie. Concentre-toi sur ta mission et profite de la soirée.</p>
    </section>{/if}
</div>

{#snippet photoActions()}
  {#if mission && proofStatus}
    {#key mission.id}
      <TreasureProof assignmentId={mission.id} status={proofStatus} busy={missionBusy} onsubmit={onproofsubmit} {oncomplete} />
    {/key}
  {/if}
{/snippet}
