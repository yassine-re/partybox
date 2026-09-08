import type { SavedSession } from "./types";

const key = (boxId: string) => `partybox:session:${boxId}`;
export function readSession(boxId: string): SavedSession | null {
  try {
    const value = JSON.parse(localStorage.getItem(key(boxId)) || "null");
    return typeof value?.token === "string" && typeof value?.gameId === "string"
      ? value
      : null;
  } catch {
    return null;
  }
}
export function saveSession(
  boxId: string,
  session: SavedSession,
  name: string,
): boolean {
  try {
    localStorage.setItem(key(boxId), JSON.stringify(session));
    localStorage.setItem("partybox:nickname", name);
    return true;
  } catch {
    return false;
  }
}
export function clearSession(boxId: string) {
  try {
    localStorage.removeItem(key(boxId));
  } catch {
    /* memory session still works */
  }
}
export function readNickname(): string {
  try {
    return localStorage.getItem("partybox:nickname") || "";
  } catch {
    return "";
  }
}
