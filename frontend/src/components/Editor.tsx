import { useEffect, useRef, useCallback } from 'react' // Added useCallback
import { EditorView, basicSetup } from 'codemirror'
import { EditorState } from '@codemirror/state'
import { StreamLanguage } from '@codemirror/language'
import { stex } from '@codemirror/legacy-modes/mode/stex'
import { autocompletion, CompletionContext } from '@codemirror/autocomplete'
import { useDebounce } from '../hooks/useDebounce' // Import useDebounce hook

interface EditorProps {
  value: string
  onChange: (value: string) => void
  onSave: () => void
}

const latexLanguage = StreamLanguage.define(stex)

function latexCompletions(context: CompletionContext) {
  const word = context.matchBefore(/\\[\w]*/)
  if (!word || (word.from === word.to && !context.explicit)) return null

  const options = [
    { label: '\\section{}', type: 'keyword' },
    { label: '\\subsection{}', type: 'keyword' },
    { label: '\\textbf{}', type: 'keyword' },
    { label: '\\textit{}', type: 'keyword' },
    { label: '\\begin{}', type: 'keyword' },
    { label: '\\end{}', type: 'keyword' },
    { label: '\\item', type: 'keyword' },
    { label: '\\label{}', type: 'keyword' },
    { label: '\\ref{}', type: 'keyword' },
    { label: '\\cite{}', type: 'keyword' },
    { label: '\\usepackage{}', type: 'keyword' },
  ]

  return {
    from: word.from,
    options,
  }
}

export default function Editor({ value, onChange, onSave }: EditorProps) {
  const editorRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)

  const debouncedOnSave = useDebounce(onSave, 10000); // 10 seconds debounce

  useEffect(() => {
    if (!editorRef.current) return

    const startState = EditorState.create({
      doc: value,
      extensions: [
        basicSetup,
        latexLanguage,
        autocompletion({ override: [latexCompletions] }),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            onChange(update.state.doc.toString())
            debouncedOnSave() // Trigger debounced save on document change
          }
        }),
        EditorView.domEventHandlers({
          keydown: (e) => {
            if (e.ctrlKey && e.key === 's') {
              e.preventDefault()
              onSave() // Immediate save on Ctrl+S
              return true
            }
            return false
          },
        }),
      ],
    })

    const view = new EditorView({
      state: startState,
      parent: editorRef.current,
    })

    viewRef.current = view

    return () => {
      view.destroy()
    }
  }, [debouncedOnSave, onChange]) // Added debouncedOnSave and onChange to dependencies

  // Update editor when value changes externally
  useEffect(() => {
    if (viewRef.current) {
      const currentValue = viewRef.current.state.doc.toString()
      if (currentValue !== value) {
        viewRef.current.dispatch({
          changes: {
            from: 0,
            to: currentValue.length,
            insert: value,
          },
        })
      }
    }
  }, [value])

  return <div ref={editorRef} className="h-full" />
}
