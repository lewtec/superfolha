import { useTranslation } from "react-i18next";

interface BinaryFileViewerProps {
  fileName: string;
  projectId: string;
}

export default function BinaryFileViewer({
  fileName,
  projectId,
}: BinaryFileViewerProps) {
  const { t } = useTranslation("editor");

  const handleDownload = () => {
    const encodedFileName = encodeURIComponent(fileName);
    const downloadUrl = `/api/projects/${projectId}/download/${encodedFileName}`;
    const a = document.createElement("a");
    a.href = downloadUrl;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  return (
    <div className="flex flex-col items-center justify-center h-full bg-base-100 text-base-content p-4">
      <p className="text-lg font-semibold mb-2">{fileName}</p>
      <p className="text-sm text-base-content/70 mb-4 text-center max-w-md">
        {t("binary_file")}
      </p>
      <button
        type="button"
        className="btn btn-primary min-h-[var(--touch-min)]"
        onClick={handleDownload}
      >
        {t("download")}
      </button>
    </div>
  );
}
