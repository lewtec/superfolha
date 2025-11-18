import { useEffect, useRef } from "react"; // Added useCallback
import { EditorView, basicSetup } from "codemirror";
import { EditorState } from "@codemirror/state";
import { StreamLanguage } from "@codemirror/language";
import { stex } from "@codemirror/legacy-modes/mode/stex";
import { autocompletion, CompletionContext } from "@codemirror/autocomplete";
import { useDebounce } from "../hooks/useDebounce"; // Import useDebounce hook

interface EditorProps {
  value: string;
  onChange: (value: string) => void;
  onSave: (content: string) => void; // Modified to accept content
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
  const isUpdatingInternally = useRef(false); // New ref to track internal updates

  const onChangeRef = useRef(onChange);
  const onSaveRef = useRef(onSave);

  useEffect(() => {
    onChangeRef.current = onChange;
    onSaveRef.current = onSave;
  }, [onChange, onSave]);

  const debouncedOnSave = useDebounce(onSaveRef.current, 1000); // 1 second debounce
  const debouncedOnSaveRef = useRef(debouncedOnSave); // New ref for debouncedOnSave

  useEffect(() => {
    debouncedOnSaveRef.current = debouncedOnSave; // Keep ref updated
  }, [debouncedOnSave]);

  useEffect(() => {
    if (!editorRef.current) return;

    const startState = EditorState.create({
      doc: value,
      extensions: [
        basicSetup,
        latexLanguage,
        autocompletion({ override: [latexCompletions] }),
        EditorView.lineWrapping, // Enable word wrap
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            const newContent = update.state.doc.toString();
            isUpdatingInternally.current = true; // Mark as internal update
            onChangeRef.current(newContent);
            debouncedOnSaveRef.current(newContent); // Pass content to debounced save
          }
        }),
        EditorView.domEventHandlers({
          keydown: (e) => {
            if (e.ctrlKey && e.key === "s") {
              e.preventDefault();
              onSaveRef.current(viewRef.current?.state.doc.toString() || ""); // Pass current content to immediate save
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
  }, []); // Empty dependency array for true one-time initialization

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
