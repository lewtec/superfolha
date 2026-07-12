import { useCallback, useEffect, useRef } from "react";
import { EditorView, basicSetup } from "codemirror";
import { EditorState } from "@codemirror/state";
import { StreamLanguage } from "@codemirror/language";
import { stex } from "@codemirror/legacy-modes/mode/stex";
import { autocompletion, CompletionContext } from "@codemirror/autocomplete";
import { useDebounce } from "../hooks/useDebounce";

interface EditorProps {
  value: string;
  onChange: (value: string) => void;
  onSave: (content: string) => void;
}

const latexLanguage = StreamLanguage.define(stex);

function latexCompletions(context: CompletionContext) {
  const word = context.matchBefore(/\\[\w]*/);
  if (!word || (word.from === word.to && !context.explicit)) return null;

  const options = [
    { label: "\\section{}", type: "keyword" },
    { label: "\\subsection{}", type: "keyword" },
    { label: "\\textbf{}", type: "keyword" },
    { label: "\\textit{}", type: "keyword" },
    { label: "\\begin{}", type: "keyword" },
    { label: "\\end{}", type: "keyword" },
    { label: "\\item", type: "keyword" },
    { label: "\\label{}", type: "keyword" },
    { label: "\\ref{}", type: "keyword" },
    { label: "\\cite{}", type: "keyword" },
    { label: "\\usepackage{}", type: "keyword" },
  ];

  return {
    from: word.from,
    options,
  };
}

export default function Editor({ value, onChange, onSave }: EditorProps) {
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const isUpdatingInternally = useRef(false);

  const onChangeRef = useRef(onChange);
  const onSaveRef = useRef(onSave);

  useEffect(() => {
    onChangeRef.current = onChange;
    onSaveRef.current = onSave;
  }, [onChange, onSave]);

  // Stable callback so useDebounce does not read ref.current during render.
  const saveViaRef = useCallback((...args: unknown[]) => {
    onSaveRef.current(String(args[0] ?? ""));
  }, []);
  const debouncedOnSave = useDebounce(saveViaRef, 1000);
  const debouncedOnSaveRef = useRef(debouncedOnSave);

  useEffect(() => {
    debouncedOnSaveRef.current = debouncedOnSave;
  }, [debouncedOnSave]);

  useEffect(() => {
    if (!editorRef.current) return;

    const startState = EditorState.create({
      doc: value,
      extensions: [
        basicSetup,
        latexLanguage,
        autocompletion({ override: [latexCompletions] }),
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const newContent = update.state.doc.toString();
            isUpdatingInternally.current = true;
            onChangeRef.current(newContent);
            debouncedOnSaveRef.current(newContent);
          }
        }),
        EditorView.domEventHandlers({
          keydown: (e) => {
            if (e.ctrlKey && e.key === "s") {
              e.preventDefault();
              onSaveRef.current(viewRef.current?.state.doc.toString() || "");
              return true;
            }
            return false;
          },
        }),
      ],
    });

    const view = new EditorView({
      state: startState,
      parent: editorRef.current,
    });

    viewRef.current = view;

    return () => {
      view.destroy();
    };
    // One-time CodeMirror mount; external value sync is handled below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update editor when value changes externally
  useEffect(() => {
    if (viewRef.current && !isUpdatingInternally.current) {
      const currentValue = viewRef.current.state.doc.toString();
      if (currentValue !== value) {
        const { selection } = viewRef.current.state; // Preserve cursor position
        viewRef.current.dispatch({
          changes: {
            from: 0,
            to: currentValue.length,
            insert: value,
          },
          selection, // Restore cursor position
          scrollIntoView: true,
        });
      }
    }
    isUpdatingInternally.current = false; // Reset flag after potential external update
  }, [value]);

  return <div ref={editorRef} className="h-full" />;
}
