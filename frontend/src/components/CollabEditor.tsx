import { useEffect, useRef } from "react";
import { EditorView, basicSetup } from "codemirror";
import { EditorState } from "@codemirror/state";
import { autocompletion } from "@codemirror/autocomplete";
import { yCollab } from "y-codemirror.next";
import * as Y from "yjs";
import type { Awareness } from "y-protocols/awareness";
import { latexCompletions, latexLanguage } from "./latexCompletions";

interface CollabEditorProps {
  ytext: Y.Text;
  awareness: Awareness;
  /** Remount when path changes */
  path: string;
}

/**
 * CodeMirror bound to a Y.Text (+ awareness carets).
 * Seeds CM from current Y.Text so remounts are not empty.
 */
export default function CollabEditor({
  ytext,
  awareness,
  path,
}: CollabEditorProps) {
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hostRef.current) return;

    const seed = ytext.toString();
    const undoManager = new Y.UndoManager(ytext);
    const view = new EditorView({
      parent: hostRef.current,
      state: EditorState.create({
        doc: seed,
        extensions: [
          basicSetup,
          latexLanguage,
          autocompletion({ override: [latexCompletions] }),
          EditorView.lineWrapping,
          yCollab(ytext, awareness, { undoManager }),
        ],
      }),
    });

    return () => {
      view.destroy();
    };
  }, [path, ytext, awareness]);

  return (
    <div
      ref={hostRef}
      className="h-full min-h-0"
      data-superfolha-editor={path}
    />
  );
}
