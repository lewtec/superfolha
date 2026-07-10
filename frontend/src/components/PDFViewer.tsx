import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

interface PDFViewerProps {
  /** Raw base64 PDF bytes, or a full data: URL */
  pdfData: string | null;
}

function toPdfBlobUrl(pdfData: string): string {
  const base64 = pdfData.startsWith("data:")
    ? pdfData.replace(/^data:application\/pdf;base64,/, "")
    : pdfData;
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  const blob = new Blob([bytes], { type: "application/pdf" });
  return URL.createObjectURL(blob);
}

/**
 * Uses the browser PDF engine (Firefox embeds Mozilla PDF.js).
 * Blob + iframe gives reliable multi-page scrolling without re-implementing the viewer.
 */
export default function PDFViewer({ pdfData }: PDFViewerProps) {
  const { t } = useTranslation("editor");
  const [error, setError] = useState<string>("");

  const blobUrl = useMemo(() => {
    if (!pdfData) return null;
    try {
      setError("");
      return toPdfBlobUrl(pdfData);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      return null;
    }
  }, [pdfData]);

  useEffect(() => {
    return () => {
      if (blobUrl) URL.revokeObjectURL(blobUrl);
    };
  }, [blobUrl]);

  if (!pdfData) {
    return (
      <div className="flex items-center justify-center h-full min-h-0 page-pad">
        <p className="text-base-content/70 text-center">
          {t("pdf_empty", {
            defaultValue:
              "No PDF compiled yet. Click Compile to generate a PDF.",
          })}
        </p>
      </div>
    );
  }

  if (error || !blobUrl) {
    return (
      <div className="flex items-center justify-center h-full min-h-0 page-pad">
        <p className="text-error text-center">{error || t("pdf_empty")}</p>
      </div>
    );
  }

  return (
    <div className="pdf-viewer h-full min-h-0 flex flex-col bg-base-200">
      <iframe
        title={t("pdf")}
        src={blobUrl}
        className="w-full flex-1 min-h-0 border-0 bg-base-100"
      />
    </div>
  );
}
