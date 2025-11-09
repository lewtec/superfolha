import { useEffect, useRef, useState } from 'react'
import * as pdfjsLib from 'pdfjs-dist'

pdfjsLib.GlobalWorkerOptions.workerSrc = `//cdnjs.cloudflare.com/ajax/libs/pdf.js/${pdfjsLib.version}/pdf.worker.min.js`

interface PDFViewerProps {
  pdfData: string | null
}

export default function PDFViewer({ pdfData }: PDFViewerProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [error, setError] = useState('')
  const [numPages, setNumPages] = useState(0)
  const [currentPage, setCurrentPage] = useState(1)
  const pdfRef = useRef<any>(null)

  useEffect(() => {
    if (!pdfData || !canvasRef.current) return

    const loadPDF = async () => {
      try {
        setError('')
        const pdfBytes = Uint8Array.from(atob(pdfData), c => c.charCodeAt(0))
        const pdf = await pdfjsLib.getDocument({ data: pdfBytes }).promise
        pdfRef.current = pdf
        setNumPages(pdf.numPages)
        renderPage(1)
      } catch (err) {
        setError('Failed to load PDF')
        console.error(err)
      }
    }

    loadPDF()
  }, [pdfData])

  const renderPage = async (pageNum: number) => {
    if (!pdfRef.current || !canvasRef.current) return

    const page = await pdfRef.current.getPage(pageNum)
    const viewport = page.getViewport({ scale: 1.5 })
    const canvas = canvasRef.current
    const context = canvas.getContext('2d')

    if (!context) return

    canvas.height = viewport.height
    canvas.width = viewport.width

    await page.render({
      canvasContext: context,
      viewport: viewport,
    }).promise

    setCurrentPage(pageNum)
  }

  if (!pdfData) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-base-content/70">No PDF compiled yet. Click "Compile" to generate PDF.</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-error">{error}</p>
      </div>
    )
  }

  return (
    <div className="pdf-viewer bg-base-200 flex flex-col h-full">
      {numPages > 1 && (
        <div className="flex justify-center items-center gap-4 p-4">
          <button
            className="btn btn-sm"
            disabled={currentPage <= 1}
            onClick={() => renderPage(currentPage - 1)}
          >
            Previous
          </button>
          <span>
            Page {currentPage} of {numPages}
          </span>
          <button
            className="btn btn-sm"
            disabled={currentPage >= numPages}
            onClick={() => renderPage(currentPage + 1)}
          >
            Next
          </button>
        </div>
      )}
      <div className="flex-1 overflow-auto flex justify-center p-4">
        <canvas ref={canvasRef} />
      </div>
    </div>
  )
}
