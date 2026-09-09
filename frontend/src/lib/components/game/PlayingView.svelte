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
  import ReactionResult from "../reaction/ReactionResult.svelte";
  import { canAssignReaction } from "$lib/reaction";

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
  }: {
    game: Game;
    players: Player[];
    playerId: string;
    mission: Mission | null;
    busy: boolean;
    realtimeStatus: RealtimeStatus;
    tab: GameTab;
    oncomplete: () => void;
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
  } = $props();
  const me = $derived(players.find((player) => player.id === playerId));
  const mode = $derived(GAME_MODES[game.mode]);
  const reactionChallenge = $derived(reactionState?.challenge ?? null);
</script>

<div class="mobile-tabs" aria-label="Affichage du jeu">
  <button
    class:active={tab === "mission"}
    aria-pressed={tab === "mission"}
    onclick={() => ontabchange("mission")}>{mode.tabLabel}</button
  ><button
    class:active={tab === "leaderboard"}
    aria-pressed={tab === "leaderboard"}
    onclick={() => ontabchange("leaderboard")}>Classement</button
  >
</div>
{#if game.mode === "chaos" && chaosState}
  <ChaosBanner state={chaosState} />
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
        {busy}
        oncomplete={oncomplete}
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
  <section
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
  </section>
</div>

{#snippet photoActions()}
  {#if mission && proofStatus}
    {#key mission.id}
      <TreasureProof assignmentId={mission.id} status={proofStatus} {busy} onsubmit={onproofsubmit} {oncomplete} />
    {/key}
  {/if}
{/snippet}
