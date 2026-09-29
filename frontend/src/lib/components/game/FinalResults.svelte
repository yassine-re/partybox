<script lang="ts">
  import { onMount } from "svelte";
  import type { Game, GameRecap, Player } from "$lib/api/types";
  import { GAME_MODES } from "$lib/game-modes";
  import PlayerList from "../PlayerList.svelte";

  let {
    game,
    players,
    playerId,
    busy,
    recap = null,
    onback,
  }: {
    game: Game;
    players: Player[];
    playerId: string;
    busy: boolean;
    recap?: GameRecap | null;
    onback: () => void;
  } = $props();
  const mode = $derived(GAME_MODES[game.mode]);
  let step = $state<"intro" | "podium" | "reveal">("intro");
  let revealTimer: ReturnType<typeof setTimeout>;
  onMount(() => {
    revealTimer = setTimeout(() => (step = "podium"), 2200);
    return () => clearTimeout(revealTimer);
  });
  const sortedPlayers = $derived([...players].sort((a, b) => (b.score ?? 0) - (a.score ?? 0)));
  const winners = $derived(sortedPlayers.filter((player) => player.score === sortedPlayers[0]?.score));
  const totalCompleted = $derived(sortedPlayers.reduce((sum, player) => sum + (player.completed_missions ?? 0), 0));
  const championAward = $derived(winners.length ? [{ icon: "🏆", title: "Champion de la soirée", player_names: winners.map((p) => p.name), description: `${sortedPlayers[0]?.score ?? 0} points au classement final.` }] : []);
  const missionLeaders = $derived.by(() => {
    if (!recap?.players.length) return [];
    const counts = recap.players.map((item) => ({ name: item.player.name, count: item.missions.filter((m) => m.status === "completed").length }));
    const best = Math.max(...counts.map((item) => item.count));
    return best > 0 ? counts.filter((item) => item.count === best).map((item) => item.name) : [];
  });
  const bigHits = $derived.by(() => {
    const completed = recap?.players.flatMap((item) => item.missions.filter((m) => m.status === "completed").map((mission) => ({ name: item.player.name, points: mission.awarded_points ?? mission.points, text: mission.text }))) ?? [];
    if (!completed.length) return null;
    const best = Math.max(...completed.map((mission) => mission.points));
    return completed.filter((mission) => mission.points === best);
  });
  const chaosEventNames = $derived((recap?.chaos_events ?? []).filter((event) => event.type === "chaos_event_triggered"));
  function chaosEventLabel(event: NonNullable<GameRecap["chaos_events"]>[number]): string {
    const details = event.payload && typeof event.payload === "object" ? event.payload as { event?: string } : {};
    return ({ double_trouble: "Double Trouble", bounty: "Bounty", mission_shuffle: "Mission Shuffle" } as Record<string, string>)[details.event ?? ""] ?? "Nouvel événement";
  }
</script>

{#if step === "intro"}
  <section class="end-intro" aria-live="polite">
    <span class="burst" aria-hidden="true">✳</span>
    <p class="eyebrow">🎉 FIN DE PARTIE</p>
    <h1>Les secrets vont<br /><em>être révélés…</em></h1>
    <div class="loading-dots" aria-label="Préparation des résultats"><i></i><i></i><i></i></div>
  </section>
{:else}
  <div class="final-layout" class:show-reveal={step === "reveal"}>
    <section class="final-feature">
      <div class="eyebrow">{mode.finalEyebrow}</div>
      <span class="final-trophy" aria-hidden="true">🏆</span>
      <h2>{winners.length > 1 ? "La victoire se partage !" : "Champion de la soirée"}<br /><em>{winners.map((player) => player.name).join(" & ")}</em></h2>
      <p>{totalCompleted} {mode.completedPlural} ensemble. Place au reveal.</p>
      <div class="final-score">{sortedPlayers[0]?.score ?? 0}<span>POINTS AU SOMMET</span></div>
      {#if step === "podium"}
        <button class="button primary" onclick={() => (step = "reveal")}>Révéler les missions <span aria-hidden="true">↓</span></button>
      {:else}
        <button class="button outline" onclick={onback} disabled={busy}>Revenir à l’accueil de la box <span aria-hidden="true">↗</span></button>
      {/if}
    </section>

    <section class="panel podium-panel">
      <div class="section-heading"><h2>Le podium</h2><span class="mini-label">CLASSEMENT FINAL</span></div>
      <div class="podium-top" aria-label="Podium final">
        {#each sortedPlayers.slice(0, 3) as player, index (player.id)}
          {@const rank = sortedPlayers.findIndex((candidate) => candidate.score === player.score) + 1}
          <div class="podium-place place-{rank}"><span>{["🥇", "🥈", "🥉"][rank - 1] ?? "🏅"}</span><strong>{player.name}</strong><b>{player.score ?? 0} pts</b><small>{player.completed_missions ?? 0} {mode.statsLabel}</small></div>
        {/each}
      </div>
      <PlayerList players={sortedPlayers} me={playerId} mode={game.mode} ranked />
    </section>

    {#if step === "reveal"}
      <section class="panel awards-panel">
        <div class="section-heading"><h2>Les awards</h2><span class="mini-label">BIEN MÉRITÉS</span></div>
        {#each championAward as award}
          <article class="award"><span>{award.icon}</span><div><strong>{award.title}</strong><p>{award.player_names.join(" & ")} — {award.description}</p></div></article>
        {/each}
        {#if missionLeaders.length}<article class="award"><span>🕵️</span><div><strong>Agent secret</strong><p>{missionLeaders.join(" & ")} — le plus de missions réussies.</p></div></article>{/if}
        {#if bigHits}<article class="award"><span>🎯</span><div><strong>Gros coup</strong>{#each bigHits as hit}<p>{hit.name} — « {hit.text} » (+{hit.points} pts).</p>{/each}</div></article>{/if}
        {#if chaosEventNames.length}<article class="award"><span>⚡</span><div><strong>Au cœur du Chaos</strong><p>{chaosEventNames.length} événement{chaosEventNames.length > 1 ? "s" : ""} Chaos déclenché{chaosEventNames.length > 1 ? "s" : ""} pendant la partie.</p><small>{chaosEventNames.slice(0, 3).map(chaosEventLabel).join(" · ")}</small></div></article>{/if}
      </section>
      <section class="mission-recap">
        <div class="section-heading"><h2>Les secrets sont dévoilés</h2><span class="mini-label">MISSION PAR JOUEUR</span></div>
        {#if recap}
          {#each recap.players as item (item.player.id)}
            <details class="player-recap"><summary><strong>{item.player.name}</strong><span>{item.missions.filter((mission) => mission.status === "completed").length} réussie(s)</span></summary>
            {#each item.missions as mission, missionIndex (missionIndex)}
                <article class:unfinished={mission.status !== "completed"}>
                  <p>{mission.text}</p>
                  {#if mission.status === "completed"}<small>✅ Réussie — +{mission.awarded_points ?? mission.points} pts</small>{:else if mission.status === "cancelled"}<small>🔀 Mission remplacée par le Chaos</small>{:else}<small>⌛ Pas terminée</small>{/if}
                </article>
              {/each}
            </details>
          {/each}
        {:else}<p class="muted">Le récapitulatif arrive…</p>{/if}
      </section>
    {/if}
  </div>
{/if}

<style>
  .end-intro { min-height: 60vh; display: grid; place-content: center; justify-items: center; text-align: center; overflow: hidden; }
  .end-intro h1 { margin: 12px 0; font-size: clamp(38px, 10vw, 72px); line-height: .98; }
  .end-intro h1 em { color: var(--lime); }
  .burst { color: var(--lime); font-size: 72px; animation: burst-in .8s cubic-bezier(.1,.8,.2,1.3) both; }
  .loading-dots { display: flex; gap: 8px; margin-top: 20px; }
  .loading-dots i { width: 7px; height: 7px; background: var(--pink); border-radius: 50%; animation: dot 1s infinite alternate; }
  .loading-dots i:nth-child(2) { animation-delay: .2s; }.loading-dots i:nth-child(3) { animation-delay: .4s; }
  .final-layout { display: grid; grid-template-columns: 1fr 1fr; gap: 22px; }
  .final-feature { padding: clamp(24px, 5vw, 48px); border-radius: 16px; background: linear-gradient(145deg, #252a1d, #171a16); border: 1px solid #414a31; }
  .final-feature h2 { font-size: clamp(26px, 5vw, 44px); line-height: 1.05; }.final-feature h2 em { color: var(--lime); }
  .final-trophy { display: block; margin-top: 22px; font-size: 62px; animation: trophy-pop .6s .2s both; }
  .final-score { margin: 22px 0; font: 900 42px monospace; color: var(--lime); }.final-score span { display: block; color: var(--muted); font: 600 10px monospace; letter-spacing: .16em; }
  .podium-top { display: grid; grid-template-columns: repeat(3, 1fr); align-items: end; gap: 7px; margin: 20px 0; }
  .podium-place { display: grid; justify-items: center; gap: 5px; padding: 12px 4px; text-align: center; border-radius: 10px 10px 4px 4px; background: #20231c; min-height: 105px; }
  .place-1 { min-height: 130px; border-top: 3px solid var(--lime); }.place-2 { border-top: 3px solid #a5a9a0; }.place-3 { border-top: 3px solid #bd8056; }
  .podium-place span { font-size: 25px; }.podium-place strong { overflow-wrap: anywhere; }.podium-place b { color: var(--lime); font-size: 13px; }.podium-place small { color: var(--muted); font-size: 10px; }
  .show-reveal { grid-template-columns: 1fr 1fr; }.awards-panel, .mission-recap { grid-column: 1 / -1; }
  .award { display: flex; gap: 12px; padding: 13px 0; border-top: 1px solid var(--line); }.award > span { font-size: 25px; }.award p { margin: 4px 0 0; color: var(--muted); }
  .player-recap { margin-top: 10px; border: 1px solid var(--line); border-radius: 10px; background: #191b18; }.player-recap summary { display: flex; justify-content: space-between; padding: 15px; cursor: pointer; }.player-recap summary span { color: var(--muted); font-size: 12px; }
  .player-recap article { padding: 12px 15px; border-top: 1px solid var(--line); }.player-recap article p { margin: 0 0 7px; }.player-recap article small { color: var(--lime); }.player-recap article.unfinished small { color: var(--muted); }
  @keyframes burst-in { from { transform: scale(.1) rotate(-90deg); opacity: 0; } to { transform: scale(1) rotate(0); opacity: 1; } }
  @keyframes trophy-pop { from { transform: translateY(18px) scale(.6); opacity: 0; } to { transform: translateY(0) scale(1); opacity: 1; } }
  @keyframes dot { to { transform: translateY(-6px); opacity: .4; } }
  @media (max-width: 740px) { .final-layout { grid-template-columns: 1fr; }.show-reveal { grid-template-columns: 1fr; } }
  @media (prefers-reduced-motion: reduce) { *, *::before, *::after { animation-duration: .01ms !important; } }
</style>
