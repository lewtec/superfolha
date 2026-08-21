import type { CompletionContext } from "@codemirror/autocomplete";
import { StreamLanguage } from "@codemirror/language";
import { stex } from "@codemirror/legacy-modes/mode/stex";

export const latexLanguage = StreamLanguage.define(stex);

export function latexCompletions(context: CompletionContext) {
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

  return { from: word.from, options };
}
