import { useState } from "react";
import { Document, Page, pdfjs } from "react-pdf";
import "react-pdf/dist/Page/AnnotationLayer.css";
import "react-pdf/dist/Page/TextLayer.css";

// Explicitly set the worker source for pdf.js
// This method is often more robust with Vite for assets within node_modules.
pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "pdfjs-dist/build/pdf.worker.min.mjs",
  import.meta.url,
).href;

interface PDFViewerProps {
  pdfData: string | null;
}

export default function PDFViewer({ pdfData }: PDFViewerProps) {
  const [numPages, setNumPages] = useState<number>(0);
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [error, setError] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(true);

  function onDocumentLoadSuccess({ numPages }: { numPages: number }) {
    setNumPages(numPages);
    setCurrentPage(1);
    setLoading(false);
    setError("");
  }

  function onDocumentLoadError(err: Error) {
    setError("Failed to load PDF: " + err.message);
    setLoading(false);
    console.error(err);
  }

  if (!pdfData) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-base-content/70">
          No PDF compiled yet. Click "Compile" to generate PDF.
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-error">{error}</p>
      </div>
    );
  }

  return (
    <div className="pdf-viewer bg-base-200 flex flex-col h-full">
      {loading && (
        <div className="flex items-center justify-center h-full">
          <span className="loading loading-spinner loading-lg"></span>
        </div>
      )}
      {!loading && numPages > 1 && (
        <div className="flex justify-center items-center gap-4 p-4">
          <button
            className="btn btn-sm"
            disabled={currentPage <= 1}
            onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
          >
            Previous
          </button>
          <span>
            Page {currentPage} of {numPages}
          </span>
          <button
            className="btn btn-sm"
            disabled={currentPage >= numPages}
            onClick={() =>
              setCurrentPage((prev) => Math.min(numPages, prev + 1))
            }
          >
            Next
          </button>
        </div>
      )}
      <div className="flex-1 overflow-auto flex justify-center p-4">
        <Document
          file={`data:application/pdf;base64,${pdfData}`}
          onLoadSuccess={onDocumentLoadSuccess}
          onLoadError={onDocumentLoadError}
          loading={
            <div className="flex items-center justify-center h-full">
              <span className="loading loading-spinner loading-lg"></span>
            </div>
          }
        >
          <Page
            pageNumber={currentPage}
            renderAnnotationLayer={true}
            renderTextLayer={true}
          />
        </Document>
      </div>
    </div>
  );
}
