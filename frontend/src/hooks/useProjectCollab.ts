import { useEffect, useState } from "react";
import { ProjectCollab, type SyncStatus } from "../collab/ProjectCollab";

export function useProjectCollab(
  projectId: string | undefined,
  email: string | null,
) {
  const [collab, setCollab] = useState<ProjectCollab | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    if (!projectId || !email) {
      return;
    }

    const session = new ProjectCollab(projectId, email);
    session.connect();
    const unsub = session.subscribe(() => setTick((n) => n + 1));
    const ready = requestAnimationFrame(() => setCollab(session));

    return () => {
      cancelAnimationFrame(ready);
      unsub();
      session.destroy();
      // Clear on next frame so we don't sync-setState in the effect body.
      requestAnimationFrame(() => setCollab((cur) => (cur === session ? null : cur)));
    };
  }, [projectId, email]);

  void tick;

  // If deps missing, treat as offline even if a previous session is lingering one frame.
  const active = projectId && email ? collab : null;

  return {
    collab: active,
    status: (active?.status ?? "offline") as SyncStatus,
    files: active?.files ?? [],
    chat: active?.chat ?? [],
    peers: active?.peers() ?? [],
  };
}
