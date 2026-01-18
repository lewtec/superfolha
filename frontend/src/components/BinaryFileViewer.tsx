// frontend/src/components/BinaryFileViewer.tsx
import React from "react";

interface BinaryFileViewerProps {
  fileName: string;
  projectId: string; // Added projectId
}

export default function BinaryFileViewer({
  fileName,
  projectId,
}: BinaryFileViewerProps) {
  const handleDownload = () => {
    // Encode the fileName to handle slashes in the path correctly
    const encodedFileName = encodeURIComponent(fileName);
    const downloadUrl = `/api/projects/${projectId}/download/${encodedFileName}`;

    console.log("Attempting to download from URL:", downloadUrl); // Debugging log

    // Inform the user about the backend dependency
    alert(
      "Download initiated. Please ensure the backend route " +
        downloadUrl +
        " is correctly implemented to serve the file.",
    );

    const a = document.createElement("a");
    a.href = downloadUrl;
    a.download = fileName; // Suggests the filename for download
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  return (
    <div
      className="
        flex h-full flex-col items-center justify-center bg-base-100 p-4
        text-base-content
      "
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="mb-4 size-24 text-warning"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
        />
      </svg>
      <p className="mb-2 text-lg font-semibold">
        Cannot display binary file: {fileName}
      </p>
      <p className="mb-4 text-sm text-base-content/70">
        This file appears to be a binary format and cannot be rendered directly
        in the editor.
      </p>
      <button className="btn btn-primary" onClick={handleDownload}>
        Download {fileName}
      </button>
    </div>
  );
}
