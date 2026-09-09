export const MISSION_FEEDBACK_FREQUENCY = 3;

export function shouldRequestMissionFeedback(
  completedMissions: number,
  alreadyCompleted: boolean,
): boolean {
  return (
    !alreadyCompleted &&
    completedMissions > 0 &&
    completedMissions % MISSION_FEEDBACK_FREQUENCY === 0
  );
}
